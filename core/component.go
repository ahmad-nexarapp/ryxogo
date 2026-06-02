// component.go — Component lifecycle for RyxoGo.
// BUG FIX #4: removed dirty flag entirely — rAF in run_wasm coalesces renders.
package core

// Component is the interface every RyxoGo component must implement.
type Component interface {
	Render() *Node
}

// Mounter is implemented by components that need to run code on mount
type Mounter interface {
	OnMount()
}

// Unmounter is implemented by components that need cleanup
type Unmounter interface {
	OnUnmount()
}

// Base is embedded in every RyxoGo component.
//
// Usage:
//
//	type Counter struct {
//	    rx.Base
//	    count *signal.Signal[int]
//	}
type Base struct{}

// Page is Base with route metadata — embed in page-level components.
//
// Usage:
//
//	type HomePage struct {
//	    rx.Page
//	}
type Page struct {
	Base
	RouteParams map[string]string // /users/:id → {"id": "123"}
	QueryParams map[string]string // ?search=foo → {"search": "foo"}
}

// Param returns a route parameter by name
func (p *Page) Param(key string) string {
	if p.RouteParams == nil {
		return ""
	}
	return p.RouteParams[key]
}

// Query returns a query parameter by name
func (p *Page) Query(key string) string {
	if p.QueryParams == nil {
		return ""
	}
	return p.QueryParams[key]
}
