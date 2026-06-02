//go:build wasm

package ryxogo

import (
	"syscall/js"

	"github.com/ahmad-nexarapp/ryxogo/core"
	"github.com/ahmad-nexarapp/ryxogo/renderer"
	"github.com/ahmad-nexarapp/ryxogo/router"
	"github.com/ahmad-nexarapp/ryxogo/signal"
)

func run(a *App) {
	var cleanup func()

	// F5 FIX: wire core.Navigate so rx.Link works without import cycle
	core.Navigate = func(path string) {
		a.router.Navigate(path)
	}

	a.router.OnChange(func(route *router.Route, params, query map[string]string) {
		if cleanup != nil {
			cleanup()
			cleanup = nil
		}
		signal.SetGlobalListener(nil)

		if route == nil { show404(a); return }
		comp := route.Handler(params, query)
		if comp == nil { show404(a); return }
		c, ok := comp.(Component)
		if !ok { return }

		type setupper interface{ Setup() }
		if s, ok := c.(setupper); ok { s.Setup() }

		r := renderer.New(a.rootID, c)

		// page holds mutable render state
		page := &pageState{r: r}
		page.init()

		cleanup = func() { page.stop() }

		if m, ok := c.(interface{ OnMount() }); ok {
			m.OnMount()
		}
	})

	js.Global().Call("addEventListener", "popstate",
		js.FuncOf(func(this js.Value, _ []js.Value) interface{} {
			path := js.Global().Get("location").Get("pathname").String()
			a.router.Navigate(path)
			return nil
		}),
	)

	currentPath := js.Global().Get("location").Get("pathname").String()
	a.router.Navigate(currentPath)
	select {}
}

// pageState holds reactive rendering state for one mounted page.
// Separating it from the closure fixes all self-reference issues.
type pageState struct {
	r            *renderer.Renderer
	stopTracking func()
	rafPending   bool
}

func (p *pageState) init() {
	// Wire async signals (fetch) to re-render
	signal.SetGlobalListener(p.scheduleUpdate)
	// First render — tracked
	p.render(true)
}

func (p *pageState) stop() {
	if p.stopTracking != nil {
		p.stopTracking()
		p.stopTracking = nil
	}
}

func (p *pageState) scheduleUpdate() {
	if p.rafPending {
		return
	}
	p.rafPending = true
	js.Global().Call("requestAnimationFrame",
		js.FuncOf(func(this js.Value, _ []js.Value) interface{} {
			p.rafPending = false
			p.render(false)
			return nil
		}),
	)
}

func (p *pageState) render(mount bool) {
	// Stop old subscriptions so we don't accumulate stale listeners
	if p.stopTracking != nil {
		p.stopTracking()
		p.stopTracking = nil
	}

	// Run render inside tracking context.
	// Any signal.Val() called during render auto-subscribes.
	// When a subscribed signal changes, scheduleUpdate fires.
	p.stopTracking = signal.Track(
		func() {
			if mount {
				p.r.Mount()
			} else {
				p.r.Update()
			}
		},
		p.scheduleUpdate,
	)
}

func show404(a *App) {
	doc := js.Global().Get("document")
	el := doc.Call("getElementById", a.rootID)
	if !el.IsNull() {
		el.Set("innerHTML", `<div style="padding:2rem;font-family:sans-serif">
			<h1>404</h1><p style="color:#666">Page not found</p>
		</div>`)
	}
}
