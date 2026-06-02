//go:build wasm

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

		comp := route.Handler(params, query)
		if comp == nil {
			show404(a)
			return
		}

		c, ok := comp.(Component)
		if !ok {
			return
		}

		// Create renderer first
		r := renderer.New(a.rootID, c)

		// scheduled tracks if a re-render is already queued
		var scheduled bool

		// scheduleUpdate batches rapid signal changes into one rAF
		scheduleUpdate := func() {
			if scheduled {
				return
			}
			scheduled = true
			js.Global().Call("requestAnimationFrame",
				js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					scheduled = false
					r.Update()
					return nil
				}),
			)
		}

		// FIX: wire signal tracking BEFORE Setup() runs
		// Any signal created in Setup() will trigger scheduleUpdate on change
		signal.SetGlobalListener(scheduleUpdate)

		// Now call Setup() — signals created here are auto-wired
		type setupper interface{ Setup() }
		if s, ok := c.(setupper); ok {
			s.Setup()
		}

		// Done wiring — clear global listener
		signal.SetGlobalListener(nil)

		// Mount into DOM
		r.Mount()

		if m, ok := c.(core_mounter); ok {
			m.OnMount()
		}
	})

	js.Global().Call("addEventListener", "popstate",
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			path := js.Global().Get("location").Get("pathname").String()
			a.router.Navigate(path)
			return nil
		}),
	)

	currentPath := js.Global().Get("location").Get("pathname").String()
	a.router.Navigate(currentPath)

	select {}
}

type core_mounter interface{ OnMount() }

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
