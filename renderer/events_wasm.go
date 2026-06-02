//go:build wasm

// events_wasm.go — stable event binding that survives VDOM patches.
//
// The naive approach (addEventListener with the Go handler directly) forces a
// full node replacement on every prop update to avoid stacking listeners —
// which loses focus, scroll position, and node identity, and is what pushes
// people toward document-level delegation workarounds.
//
// Instead, each event type is bound to an element ONCE via a stable dispatcher
// js.Func. The dispatcher looks up the element's CURRENT handler from a
// registry keyed by the element's ryxo-id. On patch we just swap the stored
// handler — the addEventListener binding never changes, so the DOM node is
// never replaced and identity/focus/scroll are preserved.
package renderer

import (
	"sync"
	"syscall/js"

	"github.com/ahmad-nexarapp/ryxogo/core"
)

// handlerSet holds the live Go handlers for one element, by event name.
type handlerSet struct {
	click     func()
	input     func(string)
	change    func(string)
	submit    func()
	focus     func()
	blur      func()
	keydown   func(string)
	keyup     func(string)
	mouseover func()
	mouseout  func()
	scroll    func(int)
	// stable dispatcher js.Funcs bound to the element (released on removal)
	bound map[string]js.Func
}

// eventRegistry maps element ryxo-id -> its live handler set.
var eventRegistry = struct {
	mu    sync.Mutex
	store map[string]*handlerSet
}{store: make(map[string]*handlerSet)}

// bindEvents attaches stable dispatchers for every event present in props and
// records the live handlers. Called when an element is first created.
func (r *Renderer) bindEvents(el js.Value, p core.Props) {
	if !hasAnyEvent(p) {
		return
	}
	id := getOrSetID(el)

	eventRegistry.mu.Lock()
	hs := eventRegistry.store[id]
	if hs == nil {
		hs = &handlerSet{bound: make(map[string]js.Func)}
		eventRegistry.store[id] = hs
	}
	eventRegistry.mu.Unlock()

	r.updateHandlers(el, hs, p)
}

// updateHandlers swaps the stored Go handlers to match new props, and ensures
// a stable dispatcher is bound for each needed event type. The dispatcher is
// bound at most once per event type per element.
func (r *Renderer) updateHandlers(el js.Value, hs *handlerSet, p core.Props) {
	hs.click = p.OnClick
	hs.input = p.OnInput
	hs.change = p.OnChange
	hs.submit = p.OnSubmit
	hs.focus = p.OnFocus
	hs.blur = p.OnBlur
	hs.keydown = p.OnKeyDown
	hs.keyup = p.OnKeyUp
	hs.mouseover = p.OnMouseOver
	hs.mouseout = p.OnMouseOut
	hs.scroll = p.OnScrollTop

	r.ensureBound(el, hs, "click", p.OnClick != nil)
	r.ensureBound(el, hs, "input", p.OnInput != nil)
	r.ensureBound(el, hs, "change", p.OnChange != nil)
	r.ensureBound(el, hs, "submit", p.OnSubmit != nil)
	r.ensureBound(el, hs, "focus", p.OnFocus != nil)
	r.ensureBound(el, hs, "blur", p.OnBlur != nil)
	r.ensureBound(el, hs, "keydown", p.OnKeyDown != nil)
	r.ensureBound(el, hs, "keyup", p.OnKeyUp != nil)
	r.ensureBound(el, hs, "mouseover", p.OnMouseOver != nil)
	r.ensureBound(el, hs, "mouseout", p.OnMouseOut != nil)
	r.ensureBound(el, hs, "scroll", p.OnScrollTop != nil)
}

// ensureBound binds a stable dispatcher for an event type if needed and not
// already bound. The dispatcher reads the live handler from hs at call time,
// so swapping hs fields is enough to change behavior — no rebinding.
func (r *Renderer) ensureBound(el js.Value, hs *handlerSet, event string, needed bool) {
	if !needed {
		return
	}
	if _, ok := hs.bound[event]; ok {
		return // already bound — handler swap already took effect
	}

	dispatcher := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		var ev js.Value
		if len(args) > 0 {
			ev = args[0]
		}
		switch event {
		case "click":
			// preventDefault on <a> stops the browser following href,
			// which would reload the whole WASM app on Link clicks.
			if this.Get("tagName").String() == "A" && !ev.IsUndefined() && !ev.IsNull() {
				ev.Call("preventDefault")
			}
			if hs.click != nil {
				hs.click()
			}
		case "input":
			if hs.input != nil {
				hs.input(this.Get("value").String())
			}
		case "change":
			if hs.change != nil {
				hs.change(this.Get("value").String())
			}
		case "submit":
			if !ev.IsUndefined() && !ev.IsNull() {
				ev.Call("preventDefault")
			}
			if hs.submit != nil {
				hs.submit()
			}
		case "focus":
			if hs.focus != nil {
				hs.focus()
			}
		case "blur":
			if hs.blur != nil {
				hs.blur()
			}
		case "keydown":
			if hs.keydown != nil {
				hs.keydown(ev.Get("key").String())
			}
		case "keyup":
			if hs.keyup != nil {
				hs.keyup(ev.Get("key").String())
			}
		case "mouseover":
			if hs.mouseover != nil {
				hs.mouseover()
			}
		case "mouseout":
			if hs.mouseout != nil {
				hs.mouseout()
			}
		case "scroll":
			if hs.scroll != nil {
				hs.scroll(this.Get("scrollTop").Int())
			}
		}
		return nil
	})

	hs.bound[event] = dispatcher
	el.Call("addEventListener", event, dispatcher)
}

// releaseEvents frees an element's stable dispatchers and registry entry.
// Called from releaseNode when the element leaves the DOM.
func releaseEvents(id string) {
	eventRegistry.mu.Lock()
	hs := eventRegistry.store[id]
	delete(eventRegistry.store, id)
	eventRegistry.mu.Unlock()
	if hs == nil {
		return
	}
	for _, f := range hs.bound {
		f.Release()
	}
}

// lookupHandlers returns the handler set for an element id, if any.
func lookupHandlers(id string) *handlerSet {
	eventRegistry.mu.Lock()
	defer eventRegistry.mu.Unlock()
	return eventRegistry.store[id]
}
