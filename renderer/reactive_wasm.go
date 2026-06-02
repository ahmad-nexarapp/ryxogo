//go:build wasm

// reactive_wasm.go — wires fine-grained bindings to signals in the browser.
//
// Three kinds of fine-grained binding, each backed by its own signal.Track
// effect so only the affected DOM updates — Render() never re-runs:
//   - reactive text   (rx.BindText)        -> updates a text node's value
//   - reactive attrs   (rx.Bindings/BindAttr/BindClass/BindStyle) -> one attr
//   - reactive show    (rx.Bindings/BindShow) -> element display:none toggle
package renderer

import (
	"sync"
	"syscall/js"

	"github.com/ahmad-nexarapp/ryxogo/core"
	"github.com/ahmad-nexarapp/ryxogo/signal"
)

// ---------------------------------------------------------
// reactive text
// ---------------------------------------------------------

// trackReactive subscribes a string compute fn to its signals. It writes the
// initial value into *initial, then on each dependency change calls
// onChange(newValue). Returns a stop function.
func trackReactive(compute func() string, onChange func(string), initial *string) func() {
	var mu sync.Mutex
	var stopFn func()

	var rerun func(first bool)
	rerun = func(first bool) {
		mu.Lock()
		if stopFn != nil {
			stopFn()
			stopFn = nil
		}
		mu.Unlock()

		var val string
		stop := signal.Track(
			func() { val = compute() },
			func() { rerun(false) },
		)

		mu.Lock()
		stopFn = stop
		mu.Unlock()

		if first {
			*initial = val
		} else {
			onChange(val)
		}
	}
	rerun(true)

	return func() {
		mu.Lock()
		if stopFn != nil {
			stopFn()
			stopFn = nil
		}
		mu.Unlock()
	}
}

// trackBool subscribes a bool compute fn to its signals, same pattern.
func trackBool(compute func() bool, onChange func(bool), initial *bool) func() {
	var mu sync.Mutex
	var stopFn func()

	var rerun func(first bool)
	rerun = func(first bool) {
		mu.Lock()
		if stopFn != nil {
			stopFn()
			stopFn = nil
		}
		mu.Unlock()

		var val bool
		stop := signal.Track(
			func() { val = compute() },
			func() { rerun(false) },
		)

		mu.Lock()
		stopFn = stop
		mu.Unlock()

		if first {
			*initial = val
		} else {
			onChange(val)
		}
	}
	rerun(true)

	return func() {
		mu.Lock()
		if stopFn != nil {
			stopFn()
			stopFn = nil
		}
		mu.Unlock()
	}
}

// ---------------------------------------------------------
// fine-grained attribute / style / visibility bindings
// ---------------------------------------------------------

// setupBindings wires each ReactiveAttr and ReactiveShow on an element to its
// own tracked effect, then registers all stop fns for cleanup on removal.
func (r *Renderer) setupBindings(el js.Value, b *core.BindingSetView) {
	var stops []func()

	// Attribute / style bindings
	for _, attr := range b.Attrs() {
		name := attr.Name
		compute := attr.Compute
		var initial string
		stop := trackReactive(compute, func(val string) {
			applyOneAttr(el, name, val)
		}, &initial)
		applyOneAttr(el, name, initial)
		stops = append(stops, stop)
	}

	// Visibility binding
	if show := b.Show(); show != nil {
		compute := show.Compute
		var initial bool
		// Remember the element's own display so we can restore it.
		prevDisplay := el.Get("style").Get("display").String()
		stop := trackBool(compute, func(visible bool) {
			applyShow(el, visible, prevDisplay)
		}, &initial)
		applyShow(el, initial, prevDisplay)
		stops = append(stops, stop)
	}

	if len(stops) > 0 {
		registerReactiveCleanup(el, func() {
			for _, s := range stops {
				s()
			}
		})
	}
}

// applyOneAttr sets a single attribute, style property, or boolean prop.
func applyOneAttr(el js.Value, name, val string) {
	switch {
	case name == "class":
		el.Set("className", val)
	case name == "value":
		el.Set("value", val)
	case name == "disabled":
		el.Set("disabled", val == "true")
	case name == "checked":
		el.Set("checked", val == "true")
	case len(name) > 6 && name[:6] == "style:":
		el.Get("style").Set(camelCase(name[6:]), val)
	default:
		el.Call("setAttribute", name, val)
	}
}

// applyShow toggles element visibility via display:none.
func applyShow(el js.Value, visible bool, prevDisplay string) {
	style := el.Get("style")
	if visible {
		if prevDisplay == "none" || prevDisplay == "" {
			style.Set("display", "")
		} else {
			style.Set("display", prevDisplay)
		}
	} else {
		style.Set("display", "none")
	}
}

// ---------------------------------------------------------
// cleanup registry (text + element bindings share this)
// ---------------------------------------------------------

var reactiveCleanups = struct {
	mu     sync.Mutex
	store  map[string]func()
	nextID uint64
}{store: make(map[string]func())}

// registerReactiveCleanup associates a stop fn with a DOM node. Text nodes
// can't hold attributes, so we stash the id as a JS property on the node.
func registerReactiveCleanup(node js.Value, stop func()) {
	reactiveCleanups.mu.Lock()
	reactiveCleanups.nextID++
	id := itoa(reactiveCleanups.nextID)
	reactiveCleanups.store[id] = stop
	reactiveCleanups.mu.Unlock()
	node.Set("__rxReactiveId", id)
}

// releaseReactive runs and removes the cleanup for a node, if any.
func releaseReactive(ref interface{}) {
	if ref == nil {
		return
	}
	el, ok := ref.(js.Value)
	if !ok {
		return
	}
	idVal := el.Get("__rxReactiveId")
	if idVal.IsNull() || idVal.IsUndefined() {
		return
	}
	id := idVal.String()
	if id == "" {
		return
	}
	reactiveCleanups.mu.Lock()
	stop := reactiveCleanups.store[id]
	delete(reactiveCleanups.store, id)
	reactiveCleanups.mu.Unlock()
	if stop != nil {
		stop()
	}
}

func itoa(n uint64) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
