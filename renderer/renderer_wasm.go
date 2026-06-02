//go:build wasm

// Package renderer — production-hardened virtual DOM renderer for RyxoGo.
//
// Fixes in this version:
//   Link preventDefault: OnClick on <a> calls event.preventDefault()
//   js.FuncOf leak: all funcs tracked in funcStore, released on node removal
//   cloneNode F2: event listeners never stack across re-renders
package renderer

import (
	"fmt"
	"strings"
	"syscall/js"

	"github.com/ahmad-nexarapp/ryxogo/core"
	"github.com/ahmad-nexarapp/ryxogo/signal"
)

type Renderer struct {
	rootEl   js.Value
	rootNode *core.Node
	comp     core.Component
}

func New(mountID string, comp core.Component) *Renderer {
	doc := js.Global().Get("document")
	el := doc.Call("getElementById", mountID)
	if el.IsNull() || el.IsUndefined() {
		panic(fmt.Sprintf("ryxogo: #%s not found", mountID))
	}
	return &Renderer{rootEl: el, comp: comp}
}

func (r *Renderer) Mount() {
	safeRender(r.rootEl, func() {
		done := signal.EnterRender()
		newTree := r.comp.Render()
		done()
		r.rootEl.Set("innerHTML", "")
		r.rootEl.Call("appendChild", r.create(newTree))
		r.rootNode = newTree
	})
}

func (r *Renderer) Update() {
	safeRender(r.rootEl, func() {
		done := signal.EnterRender()
		newTree := r.comp.Render()
		done()
		r.patch(r.rootEl, r.rootNode, newTree)
		r.rootNode = newTree
	})
}

// ---------------------------------------------------------
// create
// ---------------------------------------------------------

func (r *Renderer) create(node *core.Node) js.Value {
	if node == nil {
		return js.Global().Get("document").Call("createTextNode", "")
	}
	doc := js.Global().Get("document")
	switch node.Type {
	case core.TextNode:
		// Fine-grained reactive text: compute initial value and subscribe
		// only this node to its signals. When they change, only this text
		// node's nodeValue updates — Render() does NOT re-run.
		if rt := node.Reactive(); rt != nil {
			initial := ""
			tn := doc.Call("createTextNode", "")
			node.DOMRef = tn
			// Track the compute fn: reads inside it auto-subscribe.
			// onChange updates just this node.
			stop := trackReactive(rt.Compute, func(val string) {
				ref := node.DOMRef
				if ref != nil {
					ref.(js.Value).Set("nodeValue", val)
				}
			}, &initial)
			tn.Set("nodeValue", initial)
			// Register cleanup so the per-node effect is released on removal
			registerReactiveCleanup(tn, stop)
			return tn
		}
		tn := doc.Call("createTextNode", node.Text)
		node.DOMRef = tn
		return tn
	case core.ElementNode:
		el := doc.Call("createElement", node.Tag)
		r.applyAttrs(el, node.Props)
		r.bindEvents(el, node.Props)
		// Fine-grained attribute / style / visibility bindings.
		// Each subscribes only to the signals its compute fn reads, and
		// updates just this element — Render() never re-runs for these.
		if node.HasBindings() {
			r.setupBindings(el, node.BindingSet())
		}
		for _, child := range node.Children {
			if child != nil {
				el.Call("appendChild", r.create(child))
			}
		}
		node.DOMRef = el
		return el
	case core.FragmentNode:
		el := doc.Call("createElement", "span")
		el.Get("style").Set("display", "contents")
		for _, child := range node.Children {
			if child != nil {
				el.Call("appendChild", r.create(child))
			}
		}
		node.DOMRef = el
		return el
	}
	return doc.Call("createTextNode", "")
}

// ---------------------------------------------------------
// patch — recursive, correct parent, releases old funcs
// ---------------------------------------------------------

func (r *Renderer) patch(parent js.Value, old, new *core.Node) {
	if old == nil && new == nil {
		return
	}
	if old == nil {
		parent.Call("appendChild", r.create(new))
		return
	}
	if new == nil {
		if old.DOMRef != nil {
			el := old.DOMRef.(js.Value)
			releaseNode(old.DOMRef)      // FIX: free js.Func before removal
			parent.Call("removeChild", el)
		}
		return
	}

	// Text → Text
	if old.Type == core.TextNode && new.Type == core.TextNode {
		if old.DOMRef != nil {
			el := old.DOMRef.(js.Value)
			// Reactive text nodes manage their own value via their effect.
			// Carry the live DOM node + effect forward; don't overwrite value.
			if old.Reactive() != nil || new.Reactive() != nil {
				new.DOMRef = el
				return
			}
			if old.Text != new.Text {
				el.Set("nodeValue", new.Text)
			}
			new.DOMRef = el
		}
		return
	}

	// Different type or tag — full replace
	if old.Type != new.Type || old.Tag != new.Tag {
		newEl := r.create(new)
		if old.DOMRef != nil {
			oldEl := old.DOMRef.(js.Value)
			releaseNode(old.DOMRef)      // FIX: free js.Func before replace
			parent.Call("replaceChild", newEl, oldEl)
		} else {
			parent.Call("appendChild", newEl)
		}
		return
	}

	// Same element — update props, recurse children
	if old.DOMRef == nil {
		return
	}
	oldEl := old.DOMRef.(js.Value)
	freshEl := r.updateProps(oldEl, old.Props, new.Props)
	new.DOMRef = freshEl
	r.patchChildren(freshEl, old.Children, new.Children)
}

func (r *Renderer) patchChildren(parent js.Value, oldCh, newCh []*core.Node) {
	maxLen := len(oldCh)
	if len(newCh) > maxLen {
		maxLen = len(newCh)
	}
	for i := 0; i < maxLen; i++ {
		var old, new *core.Node
		if i < len(oldCh) { old = oldCh[i] }
		if i < len(newCh) { new = newCh[i] }
		r.patch(parent, old, new)
	}
}

// ---------------------------------------------------------
// updateProps — F2: cloneNode strips old listeners
// ---------------------------------------------------------

// updateProps updates attributes and swaps event handlers in place.
// No cloneNode, no node replacement — the element keeps its identity, focus,
// and scroll position. Stable dispatchers (events_wasm.go) mean the
// addEventListener bindings never change; we only swap the stored handlers.
func (r *Renderer) updateProps(el js.Value, old, new core.Props) js.Value {
	r.applyAttrs(el, new)
	if !hasAnyEvent(new) && !hasAnyEvent(old) {
		return el
	}
	id := getOrSetID(el)
	hs := lookupHandlers(id)
	if hs == nil {
		// Element gained events on this update — bind fresh.
		r.bindEvents(el, new)
		return el
	}
	// Swap handlers to the new props; bind any newly-needed event types.
	r.updateHandlers(el, hs, new)
	return el
}

func hasAnyEvent(p core.Props) bool {
	return p.OnClick != nil || p.OnInput != nil || p.OnChange != nil ||
		p.OnSubmit != nil || p.OnFocus != nil || p.OnBlur != nil ||
		p.OnKeyDown != nil || p.OnKeyUp != nil || p.OnMouseOver != nil ||
		p.OnMouseOut != nil || p.OnScrollTop != nil
}

// ---------------------------------------------------------
// applyAttrs — non-event attributes
// ---------------------------------------------------------

func (r *Renderer) applyAttrs(el js.Value, p core.Props) {
	if p.Class != "" { el.Set("className", p.Class) }
	if p.ID != "" { el.Set("id", p.ID) }
	if len(p.Style) > 0 {
		s := el.Get("style")
		for k, v := range p.Style { s.Set(camelCase(k), v) }
	}
	if p.Value != "" { el.Set("value", p.Value) }
	if p.Name != "" { el.Set("name", p.Name) }
	if p.Placeholder != "" { el.Set("placeholder", p.Placeholder) }
	el.Set("disabled", p.Disabled)
	if p.Checked { el.Set("checked", true) }
	if p.Required { el.Set("required", true) }
	if p.ReadOnly { el.Set("readOnly", true) }
	if p.Type != "" { el.Set("type", p.Type) }
	if p.For != "" { el.Call("setAttribute", "for", p.For) }
	if p.AutoComplete != "" { el.Call("setAttribute", "autocomplete", p.AutoComplete) }
	if p.Min != "" { el.Call("setAttribute", "min", p.Min) }
	if p.Max != "" { el.Call("setAttribute", "max", p.Max) }
	if p.Step != "" { el.Call("setAttribute", "step", p.Step) }
	if p.Pattern != "" { el.Call("setAttribute", "pattern", p.Pattern) }
	if p.Rows != "" { el.Call("setAttribute", "rows", p.Rows) }
	if p.Cols != "" { el.Call("setAttribute", "cols", p.Cols) }
	if p.Src != "" { el.Set("src", p.Src) }
	if p.Alt != "" { el.Set("alt", p.Alt) }
	if p.Href != "" { el.Set("href", p.Href) }
	if p.Target != "" { el.Set("target", p.Target) }
	for k, v := range p.Data { el.Call("setAttribute", "data-"+k, v) }
	for k, v := range p.Attrs { el.Call("setAttribute", k, v) }
}

// ---------------------------------------------------------
// attachEvents was replaced by the stable dispatcher system in events_wasm.go.
// ---------------------------------------------------------

func camelCase(s string) string {
	parts := strings.Split(s, "-")
	if len(parts) == 1 { return s }
	r2 := parts[0]
	for _, p := range parts[1:] {
		if len(p) > 0 { r2 += strings.ToUpper(p[:1]) + p[1:] }
	}
	return r2
}
