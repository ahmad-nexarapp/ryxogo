// reactive.go — fine-grained reactivity for RyxoGo.
//
// Normal rx.Text("static") renders once. The reactive helpers here create
// bindings that, when their signals change, update ONLY the affected DOM
// node/attribute/visibility — Render() does NOT re-run.
//
// This is Solid.js-style fine-grained reactivity, layered on top of the
// default coarse (whole-component) reactivity. Use it for hot paths:
// counters, live values, toggling classes, conditional visibility.
package core

// ReactiveText is a text node whose content is computed by a function.
type ReactiveText struct {
	Compute func() string
}

// ReactiveAttr binds a single attribute to a compute function.
type ReactiveAttr struct {
	Name    string // "class", "value", "href", "disabled", "style:color" etc
	Compute func() string
}

// ReactiveShow binds an element's visibility to a boolean compute function.
type ReactiveShow struct {
	Compute func() bool
}

// reactiveBindings groups all fine-grained bindings attached to an element.
type reactiveBindings struct {
	attrs []*ReactiveAttr
	show  *ReactiveShow
}

// ---------------------------------------------------------
// BindText — reactive text content
// ---------------------------------------------------------

// BindText creates a reactive text node. Only this node updates when its
// signals change — Render() does not re-run.
//
//	rx.BindText(func() string { return strconv.Itoa(p.count.Val()) })
func BindText(compute func() string) *Node {
	return &Node{
		Type:     TextNode,
		reactive: &ReactiveText{Compute: compute},
	}
}

// ---------------------------------------------------------
// Binding builder — reactive attributes, styles, visibility
// ---------------------------------------------------------

// BindingBuilder accumulates fine-grained bindings for an element.
type BindingBuilder struct {
	b *reactiveBindings
}

// Bindings starts a fine-grained binding chain for an element.
//
//	rx.Bindings().
//	    BindClass(func() string { return classFor(p.status.Val()) }).
//	    BindShow(func() bool { return p.visible.Val() }).
//	    On(rx.Div(rx.Props{Class: "card"}, children...))
func Bindings() *BindingBuilder {
	return &BindingBuilder{b: &reactiveBindings{}}
}

// BindClass reactively binds the element's class attribute.
func (bb *BindingBuilder) BindClass(compute func() string) *BindingBuilder {
	bb.b.attrs = append(bb.b.attrs, &ReactiveAttr{Name: "class", Compute: compute})
	return bb
}

// BindAttr reactively binds any named attribute (value, href, title, etc).
func (bb *BindingBuilder) BindAttr(name string, compute func() string) *BindingBuilder {
	bb.b.attrs = append(bb.b.attrs, &ReactiveAttr{Name: name, Compute: compute})
	return bb
}

// BindStyle reactively binds a single CSS style property.
//
//	.BindStyle("color", func() string { return p.color.Val() })
func (bb *BindingBuilder) BindStyle(prop string, compute func() string) *BindingBuilder {
	bb.b.attrs = append(bb.b.attrs, &ReactiveAttr{Name: "style:" + prop, Compute: compute})
	return bb
}

// BindShow reactively binds visibility (display:none when false).
//
//	.BindShow(func() bool { return p.expanded.Val() })
func (bb *BindingBuilder) BindShow(compute func() bool) *BindingBuilder {
	bb.b.show = &ReactiveShow{Compute: compute}
	return bb
}

// On attaches the accumulated bindings to an element node and returns it.
func (bb *BindingBuilder) On(node *Node) *Node {
	if node == nil {
		return nil
	}
	if node.bindings == nil {
		node.bindings = bb.b
	} else {
		node.bindings.attrs = append(node.bindings.attrs, bb.b.attrs...)
		if bb.b.show != nil {
			node.bindings.show = bb.b.show
		}
	}
	return node
}

// HasBindings reports whether a node carries fine-grained bindings.
func (n *Node) HasBindings() bool { return n.bindings != nil }

// BindingSetView is the exported, read-only view of a node's fine-grained
// bindings, consumed by the renderer (which is a separate package).
type BindingSetView struct {
	rb *reactiveBindings
}

// Attrs returns the reactive attribute bindings.
func (v *BindingSetView) Attrs() []*ReactiveAttr {
	if v.rb == nil {
		return nil
	}
	return v.rb.attrs
}

// Show returns the reactive visibility binding (may be nil).
func (v *BindingSetView) Show() *ReactiveShow {
	if v.rb == nil {
		return nil
	}
	return v.rb.show
}

// BindingSet returns an exported view of the node's bindings (may wrap nil).
func (n *Node) BindingSet() *BindingSetView {
	return &BindingSetView{rb: n.bindings}
}
