//go:build wasm

// run_wasm.go — the WASM entry point for RyxoGo apps.
// This is what actually runs in the browser.
package ryxogo

import (
	"syscall/js"

	"github.com/ahmad-nexarapp/ryxogo/renderer"
	"github.com/ahmad-nexarapp/ryxogo/router"
)

// run starts the RyxoGo app in the browser.
// Called by app.Run() when compiled to WASM.
func run(a *App) {
	// Set up the router to handle browser history
	a.router.OnChange(func(route *router.Route, params, query map[string]string) {
		if route == nil {
			show404(a)
			return
		}

		// Instantiate the page component for this route
		comp := route.Handler(params, query)
		if comp == nil {
			show404(a)
			return
		}

		c, ok := comp.(Component)
		if !ok {
			return
		}

		// Call Setup() if the component has it
		type setupper interface{ Setup() }
		if s, ok := c.(setupper); ok {
			s.Setup()
		}

		// Mount the component into the DOM
		r := renderer.New(a.rootID, c)

		// Connect re-renders to signal changes
		if b, ok := c.(interface{ SetRenderFn(func()) }); ok {
			b.SetRenderFn(func() {
				js.Global().Call("requestAnimationFrame",
					js.FuncOf(func(this js.Value, args []js.Value) interface{} {
						r.Update()
						return nil
					}),
				)
			})
		}

		r.Mount()
	})

	// Handle browser back/forward buttons
	js.Global().Call("addEventListener", "popstate",
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			path := js.Global().Get("location").Get("pathname").String()
			a.router.Navigate(path)
			return nil
		}),
	)

	// Navigate to the current URL on startup
	currentPath := js.Global().Get("location").Get("pathname").String()
	a.router.Navigate(currentPath)

	// Keep the WASM runtime alive
	select {}
}

func show404(a *App) {
	doc := js.Global().Get("document")
	el := doc.Call("getElementById", a.rootID)
	if !el.IsNull() {
		el.Set("innerHTML", `<div style="padding:2rem;font-family:sans-serif">
			<h1 style="font-size:2rem;font-weight:600">404</h1>
			<p style="color:#666">Page not found</p>
		</div>`)
	}
}
