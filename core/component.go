// component.go — Component lifecycle and base struct for RyxoGo.
package core

import "sync"

// Component is the interface every RyxoGo component must satisfy.
// Only Render() is required. Everything else is optional.
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

// Updater is implemented by components that react to prop changes
type Updater interface {
	OnUpdate()
}

// ---------------------------------------------------------
// Base — the struct developers embed in their components.
// Provides re-render scheduling and component metadata.
// ---------------------------------------------------------

// Base is embedded in every RyxoGo component.
// It gives the component a way to schedule re-renders
// and access the framework internals.
//
// Usage:
//
//	type Counter struct {
//	    ryxo.Base
//	    count *signal.Signal[int]
//	}
type Base struct {
	mu         sync.Mutex
	dirty      bool       // true when a re-render is scheduled
	renderFn   func()     // provided by the renderer
	mounted    bool
}

// ScheduleRender tells the framework this component needs to re-render.
// Called automatically by signals — developers don't call this directly.
func (b *Base) ScheduleRender() {
	b.mu.Lock()
	if b.dirty {
		b.mu.Unlock()
		return
	}
	b.dirty = true
	fn := b.renderFn
	b.mu.Unlock()

	if fn != nil {
		fn()
	}
}

// SetRenderFn is called by the renderer to connect the component to the DOM
func (b *Base) SetRenderFn(fn func()) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.renderFn = fn
}

// MarkClean resets the dirty flag after a render completes
func (b *Base) MarkClean() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.dirty = false
}

// IsMounted returns true if the component is in the DOM
func (b *Base) IsMounted() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.mounted
}

// SetMounted is called by the renderer
func (b *Base) SetMounted(v bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.mounted = v
}

// ---------------------------------------------------------
// Page — what developers embed for a full page component
// ---------------------------------------------------------

// Page is Base with route metadata.
// Embed this in page-level components (in the pages/ directory).
//
// Usage:
//
//	type HomePage struct {
//	    ryxo.Page
//	}
type Page struct {
	Base
	RouteParams map[string]string // /users/:id → {"id": "123"}
	QueryParams map[string]string // ?search=foo → {"search": "foo"}
}

// Param returns a route parameter by name
// page.Param("id") for route /users/:id
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
