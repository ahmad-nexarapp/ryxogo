// reactive.go — fine-grained reactivity for RyxoGo.
//
// Normal rx.Text("static") renders once. But rx.Bind(func() string {...})
// creates a reactive binding: when the signals it reads change, ONLY that
// text node updates — Render() does not re-run.
//
// This is Solid.js-style fine-grained reactivity, layered on top of the
// existing coarse (whole-component) reactivity. Use it for hot paths:
// counters, live values, anything that updates frequently.
package core

// ReactiveText is a text node whose content is computed by a function.
// The renderer subscribes to the signals this function reads and updates
// only this node when they change — no full component re-render.
type ReactiveText struct {
	Compute func() string // re-evaluated when dependencies change
}

// BindText creates a reactive text node.
// Only this node updates when its signals change — Render() does not re-run.
//
//	// Coarse (re-runs whole Render on change):
//	rx.Text(strconv.Itoa(p.count.Val()))
//
//	// Fine-grained (only this text node updates):
//	rx.BindText(func() string { return strconv.Itoa(p.count.Val()) })
func BindText(compute func() string) *Node {
	return &Node{
		Type:     TextNode,
		reactive: &ReactiveText{Compute: compute},
	}
}

// ReactiveAttr is a reactive attribute binding (class, style value, etc).
type ReactiveAttr struct {
	Name    string
	Compute func() string
}
