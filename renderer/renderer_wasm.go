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
		newTree := r.comp.Render()
		r.rootEl.Set("innerHTML", "")
		r.rootEl.Call("appendChild", r.create(newTree))
		r.rootNode = newTree
	})
}

func (r *Renderer) Update() {
	safeRender(r.rootEl, func() {
		newTree := r.comp.Render()
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
		tn := doc.Call("createTextNode", node.Text)
		node.DOMRef = tn
		return tn
	case core.ElementNode:
		el := doc.Call("createElement", node.Tag)
		r.applyAttrs(el, node.Props)
		r.attachEvents(el, node.Props)
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

func (r *Renderer) updateProps(el js.Value, old, new core.Props) js.Value {
	r.applyAttrs(el, new)
	if !hasAnyEvent(new) {
		return el
	}
	// Release old funcs before cloning
	releaseNode(el)

	// cloneNode(false) drops all addEventListener listeners atomically
	clone := el.Call("cloneNode", false)
	// Move children
	for {
		child := el.Get("firstChild")
		if child.IsNull() || child.IsUndefined() { break }
		clone.Call("appendChild", child)
	}
	// Swap in DOM
	parent := el.Get("parentNode")
	if !parent.IsNull() && !parent.IsUndefined() {
		parent.Call("replaceChild", clone, el)
	}
	r.attachEvents(clone, new)
	return clone
}

func hasAnyEvent(p core.Props) bool {
	return p.OnClick != nil || p.OnInput != nil || p.OnChange != nil ||
		p.OnSubmit != nil || p.OnFocus != nil || p.OnBlur != nil ||
		p.OnKeyDown != nil || p.OnKeyUp != nil || p.OnMouseOver != nil ||
		p.OnMouseOut != nil
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
	if p.Placeholder != "" { el.Set("placeholder", p.Placeholder) }
	el.Set("disabled", p.Disabled)
	if p.Checked { el.Set("checked", true) }
	if p.Type != "" { el.Set("type", p.Type) }
	if p.Src != "" { el.Set("src", p.Src) }
	if p.Alt != "" { el.Set("alt", p.Alt) }
	if p.Href != "" { el.Set("href", p.Href) }
	if p.Target != "" { el.Set("target", p.Target) }
	for k, v := range p.Data { el.Call("setAttribute", "data-"+k, v) }
	for k, v := range p.Attrs { el.Call("setAttribute", k, v) }
}

// ---------------------------------------------------------
// attachEvents — all funcs tracked in funcStore for cleanup
// ---------------------------------------------------------

func (r *Renderer) attachEvents(el js.Value, p core.Props) {
	if p.OnClick != nil {
		fn := p.OnClick
		isAnchor := el.Get("tagName").String() == "A"
		f := r.makeFunc(el, func(this js.Value, args []js.Value) interface{} {
			// Fix: preventDefault on <a> prevents browser from following href
			// This stops the WASM-reload bug on every Link click
			if isAnchor && len(args) > 0 {
				args[0].Call("preventDefault")
			}
			fn()
			return nil
		})
		el.Call("addEventListener", "click", f)
	}
	if p.OnInput != nil {
		fn := p.OnInput
		f := r.makeFunc(el, func(this js.Value, args []js.Value) interface{} {
			fn(el.Get("value").String()); return nil
		})
		el.Call("addEventListener", "input", f)
	}
	if p.OnChange != nil {
		fn := p.OnChange
		f := r.makeFunc(el, func(this js.Value, args []js.Value) interface{} {
			fn(el.Get("value").String()); return nil
		})
		el.Call("addEventListener", "change", f)
	}
	if p.OnSubmit != nil {
		fn := p.OnSubmit
		f := r.makeFunc(el, func(this js.Value, args []js.Value) interface{} {
			if len(args) > 0 { args[0].Call("preventDefault") }
			fn(); return nil
		})
		el.Call("addEventListener", "submit", f)
	}
	if p.OnFocus != nil {
		fn := p.OnFocus
		f := r.makeFunc(el, func(this js.Value, args []js.Value) interface{} { fn(); return nil })
		el.Call("addEventListener", "focus", f)
	}
	if p.OnBlur != nil {
		fn := p.OnBlur
		f := r.makeFunc(el, func(this js.Value, args []js.Value) interface{} { fn(); return nil })
		el.Call("addEventListener", "blur", f)
	}
	if p.OnKeyDown != nil {
		fn := p.OnKeyDown
		f := r.makeFunc(el, func(this js.Value, args []js.Value) interface{} {
			key := ""; if len(args) > 0 { key = args[0].Get("key").String() }
			fn(key); return nil
		})
		el.Call("addEventListener", "keydown", f)
	}
	if p.OnKeyUp != nil {
		fn := p.OnKeyUp
		f := r.makeFunc(el, func(this js.Value, args []js.Value) interface{} {
			key := ""; if len(args) > 0 { key = args[0].Get("key").String() }
			fn(key); return nil
		})
		el.Call("addEventListener", "keyup", f)
	}
	if p.OnMouseOver != nil {
		fn := p.OnMouseOver
		f := r.makeFunc(el, func(this js.Value, args []js.Value) interface{} { fn(); return nil })
		el.Call("addEventListener", "mouseover", f)
	}
	if p.OnMouseOut != nil {
		fn := p.OnMouseOut
		f := r.makeFunc(el, func(this js.Value, args []js.Value) interface{} { fn(); return nil })
		el.Call("addEventListener", "mouseout", f)
	}
}

func camelCase(s string) string {
	parts := strings.Split(s, "-")
	if len(parts) == 1 { return s }
	r2 := parts[0]
	for _, p := range parts[1:] {
		if len(p) > 0 { r2 += strings.ToUpper(p[:1]) + p[1:] }
	}
	return r2
}
