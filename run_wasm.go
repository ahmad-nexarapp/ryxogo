//go:build wasm

// run_wasm.go — WASM entry point. Fixes bugs #3, #4, #5, #11.
package ryxogo

import (
	"syscall/js"

	"github.com/ahmad-nexarapp/ryxogo/renderer"
	"github.com/ahmad-nexarapp/ryxogo/router"
	"github.com/ahmad-nexarapp/ryxogo/signal"
)

func run(a *App) {
	a.router.OnChange(func(route *router.Route, params, query map[string]string) {
		if route == nil {
			show404(a)
			return
		}

		// BUG FIX #11: call factory to get a FRESH component instance per navigation
		comp := route.Handler(params, query)
		if comp == nil {
			show404(a)
			return
		}

		c, ok := comp.(Component)
		if !ok {
			return
		}

		// Create renderer
		r := renderer.New(a.rootID, c)

		// BUG FIX #3 + #4: wire signal→render BEFORE Setup(), use rAF batching,
		// never use dirty flag (drop it entirely — rAF is the coalescing mechanism)
		var rafPending bool
		scheduleRender := func() {
			if rafPending {
				return
			}
			rafPending = true
			js.Global().Call("requestAnimationFrame",
				js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					rafPending = false
					r.Update() // BUG FIX #4: no dirty flag, always renders
					return nil
				}),
			)
		}

		// BUG FIX #3: register global listener BEFORE Setup() so every
		// signal created inside Setup() auto-triggers re-renders
		signal.SetGlobalListener(scheduleRender)

		// Run Setup() — signals created here are now auto-wired
		type setupper interface{ Setup() }
		if s, ok := c.(setupper); ok {
			s.Setup()
		}

		// Clear global listener (only used during Setup)
		signal.SetGlobalListener(nil)

		// First render
		r.Mount()

		if m, ok := c.(interface{ OnMount() }); ok {
			m.OnMount()
		}
	})

	// Browser back/forward
	js.Global().Call("addEventListener", "popstate",
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			path := js.Global().Get("location").Get("pathname").String()
			a.router.Navigate(path)
			return nil
		}),
	)

	// Initial navigation
	currentPath := js.Global().Get("location").Get("pathname").String()
	a.router.Navigate(currentPath)

	// Keep WASM alive
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
