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
	"fmt"
	"github.com/ahmad-nexarapp/ryxogo/core"
	"github.com/ahmad-nexarapp/ryxogo/signal"
	"github.com/ahmad-nexarapp/ryxogo/router"
	rhttp "github.com/ahmad-nexarapp/ryxogo/http"
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

// Persist creates a signal backed by localStorage.
// Works exactly like Use() but survives page refresh.
//
//	theme := rx.Persist("theme", "light")
//	token := rx.Persist("auth_token", "")
//	theme.Set("dark")  // saved to localStorage automatically
func Persist[T any](key string, defaultVal T) *signal.PersistSignal[T] {
	return signal.NewPersist(key, defaultVal)
}
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
	Div      = core.Div
	Span     = core.Span
	P        = core.P
	H1       = core.H1
	H2       = core.H2
	H3       = core.H3
	Button   = core.Button
	Input    = core.Input
	Form     = core.Form
	Img      = core.Img
	A        = core.A
	Ul       = core.Ul
	Ol       = core.Ol
	Li       = core.Li
	Nav      = core.Nav
	Header   = core.Header
	Main     = core.Main
	Footer   = core.Footer
	Section  = core.Section
	Article  = core.Article
	Aside    = core.Aside
	Label    = core.Label
	Textarea = core.Textarea
	Select   = core.Select
	Option   = core.Option
	Table    = core.Table
	Thead    = core.Thead
	Tbody    = core.Tbody
	Tr       = core.Tr
	Th       = core.Th
	Td       = core.Td
	Pre      = core.Pre
	Code     = core.Code
	Hr       = core.Hr
	Br       = core.Br
	Strong   = core.Strong
	Em       = core.Em
	Small    = core.Small
	Text     = core.Text
	Fragment = core.Fragment
	If       = core.If
	IfOnly   = core.IfOnly
	Nodes    = core.Nodes
	El       = core.El
)

// Bind creates Props for a controlled input from a value + setter.
// For string signals, BindString is simpler.
//
//	rx.Input(rx.Bind(p.name.Val(), func(v string) { p.name.Set(v) }, rx.Props{Class: "..."}))
var Bind = core.Bind

// BindString creates Props for a text input two-way bound to a string signal.
// Replaces manual Value + OnInput boilerplate entirely.
//
//	// Old way:
//	rx.Input(rx.Props{Value: p.name.Val(), OnInput: func(v string) { p.name.Set(v) }})
//
//	// New way:
//	rx.Input(rx.BindString(p.name, rx.Props{Placeholder: "Your name"}))
func BindString[T interface{ Val() string; Set(string) }](sig T, extra Props) Props {
	return core.BindString(sig, extra)
}

// Link renders a client-side navigation link — no page reload.
//
//	rx.Link(rx.LinkProps{To: "/about", Class: "nav-link"}, rx.Text("About"))
var Link = core.Link

// LinkProps configures a Link component.
type LinkProps = core.LinkProps

// Navigate navigates to a path client-side — no page reload.
//
//	rx.Navigate("/dashboard")
func Navigate(path string) {
	if core.Navigate != nil {
		core.Navigate(path)
	}
}

// Each renders a list from a slice — the Go equivalent of .map() in React
func Each[T any](items []T, fn func(item T, index int) *Node) []*Node {
	return core.Each(items, fn)
}

// ---------------------------------------------------------
// Store — global reactive state (F3 fix: now implemented)
// ---------------------------------------------------------

// NewStore creates a new global reactive store.
// The store holds shared state accessible from any component.
//
//	var Auth = rx.NewStore(&AuthState{})
//
//	type AuthState struct {
//	    User  *User
//	    Token string
//	}
func NewStore[T any](initial *T) *signal.Store[T] {
	return signal.NewStore(initial)
}

// GetStore reads store state and auto-subscribes for re-renders.
// Call inside Render() — when store changes, component re-renders.
//
//	user := rx.GetStore(Auth).User
func GetStore[T any](s *signal.Store[T]) *T {
	return s.Get()
}

// UpdateStore mutates store state and triggers re-renders in all subscribers.
//
//	rx.UpdateStore(Auth, func(s *AuthState) { s.User = u })
func UpdateStore[T any](s *signal.Store[T], fn func(*T)) {
	s.Update(fn)
}

// Get performs a GET request and decodes the JSON response into T.
//
//	users, err := rx.Get[[]User]("/api/users")
func Get[T any](url string) (T, error) {
	return rhttp.Get[T](url)
}

// Post performs a POST request with a JSON body.
//
//	res, err := rx.Post[Response]("/api/users", map[string]any{"name": "Alice"})
func Post[T any](url string, body interface{}) (T, error) {
	return rhttp.Post[T](url, body)
}

// Put performs a PUT request with a JSON body.
func Put[T any](url string, body interface{}) (T, error) {
	return rhttp.Put[T](url, body)
}

// Del performs a DELETE request.
func Del(url string) error {
	return rhttp.Del(url)
}

// Fetch creates a chainable HTTP request builder.
//
//	rx.Fetch("/api/users").Bearer(token).Param("page","2").DoJSON(&result)
func Fetch(url string) *rhttp.Request {
	return rhttp.Fetch(url)
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

// BasePath sets a URL prefix for subpath deploys.
// Use when your app is served at example.com/myapp/ instead of example.com/
//
//	app.BasePath("/myapp")
//	// Now routes /about → matched by browser path /myapp/about
func (a *App) BasePath(base string) *App {
	a.router.SetBasePath(base)
	return a
}

// Route registers a URL pattern with a page.
// Accepts either a component instance OR a factory function — both work:
//
//	// Simple (same instance reused — fine for stateless pages)
//	app.Route("/about", &AboutPage{})
//
//	// Factory (fresh instance per navigation — recommended for pages with state)
//	app.Route("/", func() Component { return &HomePage{} })
//
// Both signatures are supported for full backward compatibility.
func (a *App) Route(pattern string, page interface{}) *App {
	switch v := page.(type) {
	case func() Component:
		// Factory function — fresh instance per navigation
		a.router.Add(pattern, func(params, query map[string]string) interface{} {
			return v()
		})
	case Component:
		// Component instance — reused across navigations
		// Wraps it in a factory so the framework handles it uniformly
		a.router.Add(pattern, func(params, query map[string]string) interface{} {
			return v
		})
	default:
		// Unknown type — panic with a helpful message
		panic("rxgo: Route() accepts a Component or func() Component, got " +
			fmt.Sprintf("%T", page))
	}
	return a
}

// Run starts the RyxoGo application.
// In WASM: mounts to the DOM and starts listening for route changes.
// In tests: initializes the router without touching the DOM.
func (a *App) Run() {
	run(a)
}
