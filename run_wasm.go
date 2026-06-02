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

	// Enable dev-mode guards (catches Computed/Async created in Render()).
	// Cheap (one mutex check per signal creation); harmless in production.
	signal.SetDevMode(true)

	// Wire core.Navigate so rx.Link works without import cycle
	core.Navigate = func(path string) {
		a.router.Navigate(path)
	}

	// Wire core.CurrentPath so rx.Link can show active state
	core.CurrentPath = func() string {
		return a.router.Current()
	}

	a.router.OnChange(func(route *router.Route, params, query map[string]string) {
		// Call OnUnmount on previous page before switching
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

		// Activate a per-page signal scope so every Computed/Watch created
		// in Setup() is tracked and auto-Stop()ped when the page unmounts.
		scope := signal.NewScope()
		restoreScope := scope.Activate()

		type setupper interface{ Setup() }
		if s, ok := c.(setupper); ok { s.Setup() }

		restoreScope()

		r := renderer.New(a.rootID, c)

		// page holds mutable render state
		page := &pageState{r: r, comp: c}
		page.init()

		prevComp := c
		cleanup = func() {
			page.stop()
			scope.Stop() // auto-Stop all computeds/effects from Setup()
			// Call OnUnmount lifecycle hook
			if u, ok := prevComp.(interface{ OnUnmount() }); ok {
				u.OnUnmount()
			}
		}

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
	r          *renderer.Renderer
	comp       core.Component
	stopFn     func()
	rafPending bool
}

func (p *pageState) init() {
	// Wire async signals (fetch) to re-render
	signal.SetGlobalListener(p.scheduleUpdate)
	// First render — tracked
	p.render(true)
}

func (p *pageState) stop() {
	if p.stopFn != nil {
		p.stopFn()
		p.stopFn = nil
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
	if p.stopFn != nil {
		p.stopFn()
		p.stopFn = nil
	}

	p.stopFn = signal.Track(
		func() {
			if mount {
				// Hydrate if SSR content exists, otherwise fresh mount
				p.r.Hydrate()
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
