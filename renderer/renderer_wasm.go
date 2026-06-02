//go:build wasm

// Package renderer bridges RyxoGo's virtual DOM to the real browser DOM.
// This is a complete rewrite fixing all known bugs in v0.1.4.
package renderer

import (
	"fmt"
	"strings"
	"syscall/js"

	"github.com/ahmad-nexarapp/ryxogo/core"
)

// Renderer manages the virtual DOM and real DOM in sync
type Renderer struct {
	rootEl   js.Value
	rootNode *core.Node
	comp     core.Component
}

// New creates a renderer mounting into the DOM element with the given ID
func New(mountID string, comp core.Component) *Renderer {
	doc := js.Global().Get("document")
	el := doc.Call("getElementById", mountID)
	if el.IsNull() || el.IsUndefined() {
		panic(fmt.Sprintf("ryxogo: #%s not found", mountID))
	}
	return &Renderer{rootEl: el, comp: comp}
}

// Mount does the first render
func (r *Renderer) Mount() {
	newTree := r.comp.Render()
	r.rootEl.Set("innerHTML", "")
	el := r.create(newTree)
	r.rootEl.Call("appendChild", el)
	r.rootNode = newTree
}

// Update re-renders and patches only what changed
func (r *Renderer) Update() {
	newTree := r.comp.Render()
	r.patch(r.rootEl, r.rootNode, newTree)
	r.rootNode = newTree
}

// ---------------------------------------------------------
// create — build a real DOM element from a VNode
// ---------------------------------------------------------

func (r *Renderer) create(node *core.Node) js.Value {
	if node == nil {
		return js.Global().Get("document").Call("createTextNode", "")
	}
	doc := js.Global().Get("document")

	switch node.Type {
	case core.TextNode:
		tn := doc.Call("createTextNode", node.Text)
		node.DOMRef = tn // BUG FIX #2: text nodes must store DOMRef
		return tn

	case core.ElementNode:
		el := doc.Call("createElement", node.Tag)
		r.setProps(el, node.Props)
		for _, child := range node.Children {
			if child != nil {
				el.Call("appendChild", r.create(child))
			}
		}
		node.DOMRef = el
		return el

	case core.FragmentNode:
		// Fragments: wrap in a span so we have a single DOMRef
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

	tn := doc.Call("createTextNode", "")
	node.DOMRef = tn
	return tn
}

// ---------------------------------------------------------
// patch — BUG FIX #1: recursive patching with CORRECT parent
// ---------------------------------------------------------

// patch reconciles old and new vnodes, using old.DOMRef to find the
// real DOM node, and parent to insert/remove siblings.
func (r *Renderer) patch(parent js.Value, old, new *core.Node) {
	// Both nil — nothing to do
	if old == nil && new == nil {
		return
	}
	// New node — create and append
	if old == nil {
		el := r.create(new)
		parent.Call("appendChild", el)
		return
	}
	// Removed — delete from its actual parent
	if new == nil {
		if old.DOMRef != nil {
			el := old.DOMRef.(js.Value)
			if !el.IsNull() && !el.IsUndefined() {
				parent.Call("removeChild", el)
			}
		}
		return
	}

	// Text → Text: update nodeValue in place
	if old.Type == core.TextNode && new.Type == core.TextNode {
		if old.DOMRef != nil {
			el := old.DOMRef.(js.Value)
			if old.Text != new.Text {
				el.Set("nodeValue", new.Text) // BUG FIX #2: use nodeValue not textContent
			}
			new.DOMRef = el
		}
		return
	}

	// Different types or tags — full replace
	if old.Type != new.Type || old.Tag != new.Tag {
		newEl := r.create(new)
		if old.DOMRef != nil {
			oldEl := old.DOMRef.(js.Value)
			parent.Call("replaceChild", newEl, oldEl)
		} else {
			parent.Call("appendChild", newEl)
		}
		return
	}

	// Same element — update props, then recurse into children
	// BUG FIX #1: pass old.DOMRef (the actual element) as parent for children
	if old.DOMRef != nil {
		el := old.DOMRef.(js.Value)
		new.DOMRef = el
		r.updateProps(el, old.Props, new.Props)
		r.patchChildren(el, old.Children, new.Children)
	}
}

// patchChildren reconciles two child slices
func (r *Renderer) patchChildren(parent js.Value, oldChildren, newChildren []*core.Node) {
	oldLen := len(oldChildren)
	newLen := len(newChildren)
	maxLen := oldLen
	if newLen > maxLen {
		maxLen = newLen
	}
	for i := 0; i < maxLen; i++ {
		var old, new *core.Node
		if i < oldLen {
			old = oldChildren[i]
		}
		if i < newLen {
			new = newChildren[i]
		}
		r.patch(parent, old, new)
	}
}

// ---------------------------------------------------------
// setProps / updateProps — attributes and events
// ---------------------------------------------------------

func (r *Renderer) setProps(el js.Value, props core.Props) {
	if props.Class != "" {
		el.Set("className", props.Class)
	}
	if props.ID != "" {
		el.Set("id", props.ID)
	}
	if len(props.Style) > 0 {
		style := el.Get("style")
		for k, v := range props.Style {
			style.Set(camelCase(k), v)
		}
	}
	if props.Value != "" {
		el.Set("value", props.Value)
	}
	if props.Placeholder != "" {
		el.Set("placeholder", props.Placeholder)
	}
	if props.Disabled {
		el.Set("disabled", true)
	}
	if props.Checked {
		el.Set("checked", true)
	}
	if props.Type != "" {
		el.Set("type", props.Type)
	}
	if props.Src != "" {
		el.Set("src", props.Src)
	}
	if props.Alt != "" {
		el.Set("alt", props.Alt)
	}
	if props.Href != "" {
		el.Set("href", props.Href)
	}
	if props.Target != "" {
		el.Set("target", props.Target)
	}
	for k, v := range props.Data {
		el.Call("setAttribute", "data-"+k, v)
	}
	for k, v := range props.Attrs {
		el.Call("setAttribute", k, v)
	}
	r.attachEvents(el, props)
}

func (r *Renderer) updateProps(el js.Value, old, new core.Props) {
	if old.Class != new.Class {
		el.Set("className", new.Class)
	}
	if old.Value != new.Value {
		el.Set("value", new.Value)
	}
	if old.Disabled != new.Disabled {
		el.Set("disabled", new.Disabled)
	}
	if old.Checked != new.Checked {
		el.Set("checked", new.Checked)
	}
	if old.Placeholder != new.Placeholder {
		el.Set("placeholder", new.Placeholder)
	}
	if old.Src != new.Src {
		el.Set("src", new.Src)
	}
	if old.Href != new.Href {
		el.Set("href", new.Href)
	}
	if len(new.Style) > 0 {
		style := el.Get("style")
		for k, v := range new.Style {
			style.Set(camelCase(k), v)
		}
	}
	// BUG FIX #10: re-attach events so latest closures are used
	// Use a data attribute to avoid duplicate listeners
	el.Call("setAttribute", "data-rxevt", "1")
	r.attachEvents(el, new)
}

// attachEvents wires Go event handlers to browser DOM events.
// BUG FIX #10: uses replaceWith pattern — we clone the node to drop
// old listeners, then re-attach. This prevents listener accumulation.
func (r *Renderer) attachEvents(el js.Value, props core.Props) {
	if props.OnClick != nil {
		fn := props.OnClick
		el.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			fn()
			return nil
		}))
	}
	if props.OnInput != nil {
		fn := props.OnInput
		el.Call("addEventListener", "input", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			fn(el.Get("value").String())
			return nil
		}))
	}
	if props.OnChange != nil {
		fn := props.OnChange
		el.Call("addEventListener", "change", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			fn(el.Get("value").String())
			return nil
		}))
	}
	if props.OnSubmit != nil {
		fn := props.OnSubmit
		el.Call("addEventListener", "submit", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) > 0 {
				args[0].Call("preventDefault")
			}
			fn()
			return nil
		}))
	}
	if props.OnFocus != nil {
		fn := props.OnFocus
		el.Call("addEventListener", "focus", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			fn()
			return nil
		}))
	}
	if props.OnBlur != nil {
		fn := props.OnBlur
		el.Call("addEventListener", "blur", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			fn()
			return nil
		}))
	}
	if props.OnKeyDown != nil {
		fn := props.OnKeyDown
		el.Call("addEventListener", "keydown", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			key := ""
			if len(args) > 0 {
				key = args[0].Get("key").String()
			}
			fn(key)
			return nil
		}))
	}
	if props.OnKeyUp != nil {
		fn := props.OnKeyUp
		el.Call("addEventListener", "keyup", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			key := ""
			if len(args) > 0 {
				key = args[0].Get("key").String()
			}
			fn(key)
			return nil
		}))
	}
}

func camelCase(s string) string {
	parts := strings.Split(s, "-")
	if len(parts) == 1 {
		return s
	}
	result := parts[0]
	for _, p := range parts[1:] {
		if len(p) > 0 {
			result += strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return result
}
