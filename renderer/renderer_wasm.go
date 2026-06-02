//go:build wasm

// Package renderer bridges RyxoGo's virtual DOM to the real browser DOM.
// This file only compiles when targeting WebAssembly (GOARCH=wasm GOOS=js).
// It uses Go's syscall/js package to call browser APIs directly.
package renderer

import (
	"fmt"
	"strings"
	"syscall/js"

	"github.com/ahmad-nexarapp/ryxogo/core"
)

// ---------------------------------------------------------
// Renderer — manages the root DOM mount and re-renders
// ---------------------------------------------------------

// Renderer mounts a component tree into a real DOM element
type Renderer struct {
	rootEl   js.Value   // the real DOM element we mount into
	rootNode *core.Node // last rendered virtual DOM tree
	comp     core.Component
}

// New creates a renderer that mounts into the DOM element with the given ID
func New(mountID string, comp core.Component) *Renderer {
	doc := js.Global().Get("document")
	el := doc.Call("getElementById", mountID)
	if el.IsNull() || el.IsUndefined() {
		panic(fmt.Sprintf("ryxogo: mount element #%s not found", mountID))
	}
	return &Renderer{
		rootEl: el,
		comp:   comp,
	}
}

// Mount performs the first render and mounts into the DOM
func (r *Renderer) Mount() {
	newTree := r.comp.Render()
	r.rootEl.Set("innerHTML", "") // clear existing content
	el := r.createElement(newTree)
	r.rootEl.Call("appendChild", el)
	r.rootNode = newTree

	// Call OnMount if implemented
	if m, ok := r.comp.(core.Mounter); ok {
		m.OnMount()
	}
}

// Update diffs old and new virtual DOM and patches the real DOM
func (r *Renderer) Update() {
	newTree := r.comp.Render()
	patches := core.Diff(r.rootNode, newTree)
	r.applyPatches(r.rootEl, patches)
	r.rootNode = newTree
}

// ---------------------------------------------------------
// createElement — turns a virtual DOM node into a real DOM element
// ---------------------------------------------------------

func (r *Renderer) createElement(node *core.Node) js.Value {
	if node == nil {
		return js.Null()
	}

	doc := js.Global().Get("document")

	switch node.Type {
	case core.TextNode:
		return doc.Call("createTextNode", node.Text)

	case core.FragmentNode:
		frag := doc.Call("createDocumentFragment")
		for _, child := range node.Children {
			if child != nil {
				frag.Call("appendChild", r.createElement(child))
			}
		}
		return frag

	case core.ElementNode:
		el := doc.Call("createElement", node.Tag)
		r.applyProps(el, node.Props)

		for _, child := range node.Children {
			if child != nil {
				el.Call("appendChild", r.createElement(child))
			}
		}

		// Store reference back to real DOM element
		node.DOMRef = el
		return el
	}

	return doc.Call("createTextNode", "")
}

// ---------------------------------------------------------
// applyProps — sets attributes, classes, styles, event handlers
// ---------------------------------------------------------

func (r *Renderer) applyProps(el js.Value, props core.Props) {
	// Class
	if props.Class != "" {
		el.Set("className", props.Class)
	}

	// ID
	if props.ID != "" {
		el.Set("id", props.ID)
	}

	// Inline styles
	if len(props.Style) > 0 {
		style := el.Get("style")
		for k, v := range props.Style {
			style.Set(camelCase(k), v)
		}
	}

	// Standard attributes
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

	// Data attributes
	for k, v := range props.Data {
		el.Call("setAttribute", "data-"+k, v)
	}

	// Extra attributes
	for k, v := range props.Attrs {
		el.Call("setAttribute", k, v)
	}

	// Event handlers — wrapped in js.Func to prevent GC
	if props.OnClick != nil {
		fn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			props.OnClick()
			return nil
		})
		el.Call("addEventListener", "click", fn)
	}

	if props.OnInput != nil {
		fn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			val := el.Get("value").String()
			props.OnInput(val)
			return nil
		})
		el.Call("addEventListener", "input", fn)
	}

	if props.OnChange != nil {
		fn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			val := el.Get("value").String()
			props.OnChange(val)
			return nil
		})
		el.Call("addEventListener", "change", fn)
	}

	if props.OnSubmit != nil {
		fn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) > 0 {
				args[0].Call("preventDefault")
			}
			props.OnSubmit()
			return nil
		})
		el.Call("addEventListener", "submit", fn)
	}

	if props.OnFocus != nil {
		fn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			props.OnFocus()
			return nil
		})
		el.Call("addEventListener", "focus", fn)
	}

	if props.OnBlur != nil {
		fn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			props.OnBlur()
			return nil
		})
		el.Call("addEventListener", "blur", fn)
	}

	if props.OnKeyDown != nil {
		fn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			key := ""
			if len(args) > 0 {
				key = args[0].Get("key").String()
			}
			props.OnKeyDown(key)
			return nil
		})
		el.Call("addEventListener", "keydown", fn)
	}

	if props.OnKeyUp != nil {
		fn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			key := ""
			if len(args) > 0 {
				key = args[0].Get("key").String()
			}
			props.OnKeyUp(key)
			return nil
		})
		el.Call("addEventListener", "keyup", fn)
	}

	if props.OnMouseOver != nil {
		fn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			props.OnMouseOver()
			return nil
		})
		el.Call("addEventListener", "mouseover", fn)
	}

	if props.OnMouseOut != nil {
		fn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			props.OnMouseOut()
			return nil
		})
		el.Call("addEventListener", "mouseout", fn)
	}
}

// ---------------------------------------------------------
// applyPatches — applies diff patches to the real DOM
// ---------------------------------------------------------

func (r *Renderer) applyPatches(parent js.Value, patches []core.Patch) {
	for _, patch := range patches {
		switch patch.Type {

		case core.PatchCreate:
			el := r.createElement(patch.NewNode)
			parent.Call("appendChild", el)

		case core.PatchRemove:
			if patch.OldNode != nil && patch.OldNode.DOMRef != nil {
				el := patch.OldNode.DOMRef.(js.Value)
				if !el.IsNull() && !el.IsUndefined() {
					parent.Call("removeChild", el)
				}
			}

		case core.PatchReplace:
			if patch.OldNode != nil && patch.OldNode.DOMRef != nil {
				oldEl := patch.OldNode.DOMRef.(js.Value)
				newEl := r.createElement(patch.NewNode)
				parent.Call("replaceChild", newEl, oldEl)
			}

		case core.PatchUpdate:
			if patch.OldNode != nil && patch.OldNode.DOMRef != nil {
				el := patch.OldNode.DOMRef.(js.Value)
				patch.NewNode.DOMRef = el
				r.updateProps(el, patch.OldNode.Props, patch.NewNode.Props)
			}

		case core.PatchText:
			if patch.OldNode != nil && patch.OldNode.DOMRef != nil {
				el := patch.OldNode.DOMRef.(js.Value)
				el.Set("textContent", patch.NewNode.Text)
				patch.NewNode.DOMRef = el
			}
		}
	}
}

// updateProps efficiently updates only changed props on an existing element
func (r *Renderer) updateProps(el js.Value, old, new core.Props) {
	if old.Class != new.Class {
		el.Set("className", new.Class)
	}
	if old.ID != new.ID {
		el.Set("id", new.ID)
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

	// Update styles
	if len(new.Style) > 0 {
		style := el.Get("style")
		for k, v := range new.Style {
			style.Set(camelCase(k), v)
		}
	}

	// Re-attach event handlers (always update to latest closures)
	r.applyProps(el, new)
}

// camelCase converts CSS property names to JS camelCase
// "background-color" → "backgroundColor"
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
