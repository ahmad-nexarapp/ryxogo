//go:build wasm

// Package renderer bridges RyxoGo's virtual DOM to the real browser DOM.
package renderer

import (
	"fmt"
	"strings"
	"syscall/js"

	"github.com/ahmad-nexarapp/ryxogo/core"
)

// Renderer manages the root DOM mount and re-renders
type Renderer struct {
	rootEl    js.Value
	rootNode  *core.Node
	comp      core.Component
	rendering bool // guard against concurrent renders
}

// New creates a renderer that mounts into the DOM element with the given ID
func New(mountID string, comp core.Component) *Renderer {
	doc := js.Global().Get("document")
	el := doc.Call("getElementById", mountID)
	if el.IsNull() || el.IsUndefined() {
		panic(fmt.Sprintf("ryxogo: mount element #%s not found", mountID))
	}
	return &Renderer{rootEl: el, comp: comp}
}

// Mount performs the first render
func (r *Renderer) Mount() {
	newTree := r.comp.Render()
	r.rootEl.Set("innerHTML", "")
	el := r.createElement(newTree)
	r.rootEl.Call("appendChild", el)
	r.rootNode = newTree

	if m, ok := r.comp.(core.Mounter); ok {
		m.OnMount()
	}
}

// Update diffs and patches the DOM — safe to call from requestAnimationFrame
func (r *Renderer) Update() {
	if r.rendering {
		return
	}
	r.rendering = true
	defer func() { r.rendering = false }()

	newTree := r.comp.Render()
	r.patchChildren(r.rootEl, r.rootNode.Children, newTree.Children)
	// Copy DOMRefs from old tree so future patches work
	copyDOMRefs(r.rootNode, newTree)
	r.rootNode = newTree
}

// copyDOMRefs copies DOM references from old tree to new tree after a diff
func copyDOMRefs(old, new *core.Node) {
	if old == nil || new == nil {
		return
	}
	if old.DOMRef != nil && new.DOMRef == nil {
		new.DOMRef = old.DOMRef
	}
	min := len(old.Children)
	if len(new.Children) < min {
		min = len(new.Children)
	}
	for i := 0; i < min; i++ {
		copyDOMRefs(old.Children[i], new.Children[i])
	}
}

// ---------------------------------------------------------
// createElement — turns a VNode into a real DOM element
// ---------------------------------------------------------

func (r *Renderer) createElement(node *core.Node) js.Value {
	if node == nil {
		return js.Null()
	}
	doc := js.Global().Get("document")

	switch node.Type {
	case core.TextNode:
		el := doc.Call("createTextNode", node.Text)
		node.DOMRef = el // FIX: store DOMRef for text nodes
		return el

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
				childEl := r.createElement(child)
				el.Call("appendChild", childEl)
			}
		}
		node.DOMRef = el // store DOMRef for element nodes
		return el
	}

	return doc.Call("createTextNode", "")
}

// ---------------------------------------------------------
// Patch — smart update without full re-create
// ---------------------------------------------------------

// patchNode updates a single node in place when possible
func (r *Renderer) patchNode(parent js.Value, old, new *core.Node, index int) {
	if old == nil && new == nil {
		return
	}

	// New node — append it
	if old == nil {
		el := r.createElement(new)
		parent.Call("appendChild", el)
		return
	}

	// Removed — delete it
	if new == nil {
		if old.DOMRef != nil {
			el := old.DOMRef.(js.Value)
			if !el.IsNull() && !el.IsUndefined() {
				parent.Call("removeChild", el)
			}
		}
		return
	}

	// Both text nodes
	if old.Type == core.TextNode && new.Type == core.TextNode {
		if old.Text != new.Text && old.DOMRef != nil {
			el := old.DOMRef.(js.Value)
			el.Set("nodeValue", new.Text) // FIX: use nodeValue for text nodes
			new.DOMRef = el
		} else {
			new.DOMRef = old.DOMRef
		}
		return
	}

	// Different types or different tags — replace entirely
	if old.Type != new.Type || old.Tag != new.Tag {
		newEl := r.createElement(new)
		if old.DOMRef != nil {
			oldEl := old.DOMRef.(js.Value)
			parent.Call("replaceChild", newEl, oldEl)
		} else {
			parent.Call("appendChild", newEl)
		}
		return
	}

	// Same element — update props and recurse into children
	if old.DOMRef != nil {
		el := old.DOMRef.(js.Value)
		new.DOMRef = el
		r.updateProps(el, old.Props, new.Props)
		r.patchChildren(el, old.Children, new.Children)
	}
}

// patchChildren reconciles two child lists
func (r *Renderer) patchChildren(parent js.Value, oldChildren, newChildren []*core.Node) {
	maxLen := len(oldChildren)
	if len(newChildren) > maxLen {
		maxLen = len(newChildren)
	}
	for i := 0; i < maxLen; i++ {
		var old, new *core.Node
		if i < len(oldChildren) {
			old = oldChildren[i]
		}
		if i < len(newChildren) {
			new = newChildren[i]
		}
		r.patchNode(parent, old, new, i)
	}
}

// ---------------------------------------------------------
// applyProps — set attrs, styles, events on a DOM element
// ---------------------------------------------------------

func (r *Renderer) applyProps(el js.Value, props core.Props) {
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

// updateProps updates only changed props on an existing element
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
	// Always re-attach event handlers so closures capture latest state
	r.attachEvents(el, new)
}

// attachEvents wires Go functions to DOM events
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
