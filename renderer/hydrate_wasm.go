//go:build wasm

// hydrate_wasm.go — hydration attaches event handlers to server-rendered DOM
// without destroying and recreating it. This avoids the flash of blank content
// and preserves SSR's fast first paint.
package renderer

import (
	"syscall/js"

	"github.com/ahmad-nexarapp/ryxogo/core"
)

// Hydrate attaches the component to existing server-rendered DOM.
// Unlike Mount() which wipes innerHTML, Hydrate walks the existing DOM
// and attaches event handlers in place.
func (r *Renderer) Hydrate() {
	// Check if there's server-rendered content
	ssr := r.rootEl.Call("getAttribute", "data-ssr")
	if ssr.IsNull() || ssr.IsUndefined() || ssr.String() != "true" {
		// No SSR content — fall back to normal mount
		r.Mount()
		return
	}

	safeRender(r.rootEl, func() {
		newTree := r.comp.Render()

		// Walk the existing DOM and attach event handlers + DOMRefs
		firstChild := r.rootEl.Get("firstElementChild")
		if firstChild.IsNull() || firstChild.IsUndefined() {
			// No existing DOM — mount fresh
			r.rootEl.Set("innerHTML", "")
			r.rootEl.Call("appendChild", r.create(newTree))
		} else {
			// Hydrate: bind virtual DOM to existing real DOM
			r.hydrateNode(firstChild, newTree)
		}

		r.rootNode = newTree

		// Clear the SSR marker now that we've hydrated
		r.rootEl.Call("removeAttribute", "data-ssr")
	})
}

// hydrateNode binds a virtual node to an existing real DOM node,
// attaching event handlers without recreating elements.
func (r *Renderer) hydrateNode(el js.Value, node *core.Node) {
	if node == nil || el.IsNull() || el.IsUndefined() {
		return
	}

	switch node.Type {
	case core.TextNode:
		node.DOMRef = el

	case core.ElementNode:
		node.DOMRef = el
		// Attach event handlers to the existing element
		r.attachEvents(el, node.Props)

		// Recurse into children
		childNodes := el.Get("childNodes")
		childCount := childNodes.Get("length").Int()
		vChildIdx := 0
		for i := 0; i < childCount && vChildIdx < len(node.Children); i++ {
			realChild := childNodes.Index(i)
			nodeType := realChild.Get("nodeType").Int()
			// nodeType 3 = text, 1 = element
			if nodeType == 3 || nodeType == 1 {
				vChild := node.Children[vChildIdx]
				if vChild != nil {
					r.hydrateNode(realChild, vChild)
					vChildIdx++
				}
			}
		}

	case core.FragmentNode:
		// Fragments rendered as span wrapper in SSR
		node.DOMRef = el
		for i, child := range node.Children {
			childEl := el.Get("childNodes").Index(i)
			if !childEl.IsNull() && !childEl.IsUndefined() {
				r.hydrateNode(childEl, child)
			}
		}
	}
}
