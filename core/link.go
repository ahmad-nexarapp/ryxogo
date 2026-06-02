// link.go — client-side navigation. Prevents full page reload on link clicks.
package core

// Navigate is wired by run_wasm.go to the router's Navigate fn.
var Navigate func(path string)

// LinkProps configures a Link component.
type LinkProps struct {
	To     string
	Class  string
	Style  map[string]string
	Attrs  map[string]string
	Active string // class to add when To matches current path
}

// CurrentPath is wired by run_wasm.go so Link can detect active route.
var CurrentPath func() string

// Link renders an <a> tag that navigates client-side without a page reload.
// It calls preventDefault() on the click event so the browser never follows
// the href — solving the WASM-reload bug in v0.1.9.
//
//	rx.Link(rx.LinkProps{To: "/about", Class: "nav-link"}, rx.Text("About"))
func Link(props LinkProps, children ...*Node) *Node {
	class := props.Class

	// Add active class if current path matches
	if props.Active != "" && CurrentPath != nil {
		if CurrentPath() == props.To {
			if class != "" {
				class += " " + props.Active
			} else {
				class = props.Active
			}
		}
	}

	to := props.To
	return El("a", Props{
		Href:  to,
		Class: class,
		Style: props.Style,
		Attrs: props.Attrs,
		// preventDefault stops browser from following href (fixes WASM reload bug)
		OnClick: func() {
			if Navigate != nil {
				Navigate(to)
			}
		},
	}, children...)
}
