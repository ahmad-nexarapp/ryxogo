// Package ryxogo is the main entry point for the RyxoGo framework.
//
// RyxoGo is a Go-first frontend framework that compiles to WebAssembly.
// It brings Go's speed, type safety, and concurrency to the browser.
//
// Quick start:
//
//	type Counter struct{ ryxogo.Page }
//
//	func (c *Counter) Setup() {
//	    c.count = ryxogo.Use(0)
//	}
//
//	func (c *Counter) Render() *ryxogo.Node {
//	    return ryxogo.Div(ryxogo.P{},
//	        ryxogo.P(ryxogo.P{}, ryxogo.Text(c.count.Val())),
//	        ryxogo.Button(ryxogo.P{OnClick: func() { c.count.Set(c.count.Val()+1) }},
//	            ryxogo.Text("Click me"),
//	        ),
//	    )
//	}
//
//	func main() {
//	    app := ryxogo.New()
//	    app.Route("/", &Counter{})
//	    app.Run()
//	}
package ryxogo

import (
	"github.com/ahmad-nexarapp/ryxogo/core"
	"github.com/ahmad-nexarapp/ryxogo/signal"
	"github.com/ahmad-nexarapp/ryxogo/router"
)

// ---------------------------------------------------------
// Re-export core types so developers only import "ryxogo"
// ---------------------------------------------------------

// Node is a virtual DOM node
type Node = core.Node

// Props holds all element attributes and event handlers
type Props = core.Props

// Component is the interface all RyxoGo components implement
type Component = core.Component

// Page is the base struct for page-level components
type Page = core.Page

// Base is the base struct for all components
type Base = core.Base

// State values for AsyncSignal
const (
	Loading = signal.Loading
	Success = signal.Success
	Error   = signal.Error
)

// ---------------------------------------------------------
// Signal API — the state system
// ---------------------------------------------------------

// Use creates a new reactive signal with an initial value.
// The component re-renders automatically when the value changes.
//
//	count := ryxogo.Use(0)
//	count.Val()        // read
//	count.Set(5)       // write → triggers re-render
//	count.Update(func(v int) int { return v + 1 })
func Use[T any](initial T) *signal.Signal[T] {
	return signal.New(initial)
}

// Computed creates a derived signal that updates automatically
// when any signal it reads changes. No dependency array needed.
//
//	total := ryxogo.Computed(func() float64 {
//	    return price.Val() * float64(qty.Val())
//	})
func Computed[T any](fn func() T) *signal.Computed[T] {
	return signal.Derive(fn)
}

// Watch runs a function whenever its signal dependencies change.
// RyxoGo automatically detects which signals the function reads.
//
//	ryxogo.Watch(func() {
//	    fmt.Println("name changed:", name.Val())
//	})
func Watch(fn func()) *signal.Effect {
	return signal.Watch(fn)
}

// Async creates an async signal that fetches data and tracks
// loading, success, and error states automatically.
//
//	users := ryxogo.Async(func() ([]User, error) {
//	    return ryxogo.Get[[]User]("/api/users")
//	})
func Async[T any](fn func() (T, error)) *signal.AsyncSignal[T] {
	return signal.Async(fn)
}

// ---------------------------------------------------------
// Element builder API
// ---------------------------------------------------------

// Re-export all element builders
var (
	Div     = core.Div
	Span    = core.Span
	P       = core.P
	H1      = core.H1
	H2      = core.H2
	H3      = core.H3
	Button  = core.Button
	Input   = core.Input
	Form    = core.Form
	Img     = core.Img
	A       = core.A
	Ul      = core.Ul
	Li      = core.Li
	Nav     = core.Nav
	Header  = core.Header
	Main    = core.Main
	Footer  = core.Footer
	Section = core.Section
	Article = core.Article
	Text    = core.Text
	Fragment = core.Fragment
	If      = core.If
	IfOnly  = core.IfOnly
	Nodes   = core.Nodes
)

// Each renders a list from a slice — the Go equivalent of .map() in React
func Each[T any](items []T, fn func(item T, index int) *Node) []*Node {
	return core.Each(items, fn)
}

// ---------------------------------------------------------
// App — the entry point
// ---------------------------------------------------------

// App is the root RyxoGo application
type App struct {
	router  *router.Router
	rootID  string
}

// New creates a new RyxoGo application
func New() *App {
	return &App{
		router: router.New(),
		rootID: "app", // mounts into <div id="app">
	}
}

// MountTo sets the DOM element ID to mount into (default: "app")
func (a *App) MountTo(id string) *App {
	a.rootID = id
	return a
}

// Route registers a page component for a URL pattern
//
//	app.Route("/", &HomePage{})
//	app.Route("/users/:id", &UserPage{})
//	app.Route("/blog/:slug", &BlogPost{})
func (a *App) Route(pattern string, component Component) *App {
	a.router.Add(pattern, func(params, query map[string]string) interface{} {
		return component
	})
	return a
}

// Run starts the RyxoGo application.
// In WASM: mounts to the DOM and starts listening for route changes.
// In tests: initializes the router without touching the DOM.
func (a *App) Run() {
	run(a)
}
