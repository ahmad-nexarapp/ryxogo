// link.go — client-side navigation helpers.
// F5 FIX: prevents full page reloads when navigating between routes.
package core

// Navigate is set by run_wasm.go to the router's Navigate function.
// This avoids import cycles between core and router packages.
var Navigate func(path string)

// LinkProps for the Link component
type LinkProps struct {
	To      string // destination path e.g. "/about"
	Class   string
	Style   map[string]string
	Attrs   map[string]string
}

// Link renders a client-side navigation link.
// Prevents full page reload — uses router.Navigate instead.
//
//	rx.Link(rx.LinkProps{To: "/about", Class: "nav-link"}, rx.Text("About"))
func Link(props LinkProps, children ...*Node) *Node {
	// Build onclick that intercepts navigation
	var clickFn func()
	if Navigate != nil {
		to := props.To
		clickFn = func() { Navigate(to) }
	}

	return El("a", Props{
		Href:    props.To,
		Class:   props.Class,
		Style:   props.Style,
		Attrs:   props.Attrs,
		OnClick: clickFn,
	}, children...)
}
