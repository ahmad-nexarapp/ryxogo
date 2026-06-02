// lazy.go — lazy route loading.
//
// Go WASM compiles to a single binary — true code splitting isn't possible
// like JS dynamic import(). But we CAN defer expensive work:
//   - Component Setup() only runs when the route is first visited
//   - Heavy data fetching is deferred until mount
//   - Route components are created on demand via factory functions
//
// This gives the practical benefit of code splitting (faster initial render,
// work done only when needed) within Go WASM's single-binary constraint.
package ryxogo

import "github.com/ahmad-nexarapp/ryxogo/core"

// LazyComponent wraps a factory that's only called when the route is visited.
type LazyComponent struct {
	core.Base
	factory  func() core.Component
	loaded   core.Component
	loading  bool
	fallback *core.Node
}

// Lazy creates a lazily-initialized route component.
// The factory only runs when the user first navigates to this route.
//
//	app.Route("/heavy", rx.Lazy(func() rx.Component {
//	    return &HeavyDashboard{}  // only constructed on first visit
//	}))
func Lazy(factory func() core.Component) func() core.Component {
	return func() core.Component {
		return &LazyComponent{
			factory: factory,
			fallback: core.Div(core.Props{Class: "rxui-lazy-loading"},
				core.Text("Loading...")),
		}
	}
}

// LazyWithFallback is like Lazy but with a custom loading UI.
func LazyWithFallback(factory func() core.Component, fallback *core.Node) func() core.Component {
	return func() core.Component {
		return &LazyComponent{
			factory:  factory,
			fallback: fallback,
		}
	}
}

// Setup defers to the wrapped component's Setup on first render.
func (l *LazyComponent) Setup() {
	if l.loaded == nil {
		l.loaded = l.factory()
		if s, ok := l.loaded.(interface{ Setup() }); ok {
			s.Setup()
		}
	}
}

// Render delegates to the loaded component.
func (l *LazyComponent) Render() *core.Node {
	if l.loaded == nil {
		return l.fallback
	}
	return l.loaded.Render()
}
