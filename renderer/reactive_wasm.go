//go:build wasm

// reactive_wasm.go — wires fine-grained reactive text nodes to signals.
//
// When a BindText node is created, trackReactive subscribes the compute fn
// to whatever signals it reads. On change, it re-runs compute, pushes the new
// string to the single DOM text node, and re-tracks (deps may have changed).
// Render() is never re-run — only the one text node updates. This is the
// Solid.js fine-grained model, available alongside the default coarse model.
package renderer

import (
	"sync"
	"syscall/js"

	"github.com/ahmad-nexarapp/ryxogo/signal"
)

// trackReactive subscribes compute to its signals. It writes the initial
// computed value into *initial, then on every dependency change calls
// onChange(newValue). Returns a stop function that cancels the subscription.
func trackReactive(compute func() string, onChange func(string), initial *string) func() {
	// stopFn holds the current Track cleanup; reassigned on each re-track.
	var mu sync.Mutex
	var stopFn func()

	// rerun re-evaluates compute under tracking and pushes the value out.
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
			func() { val = compute() }, // reads here auto-subscribe
			func() {
				// A dependency changed — re-evaluate and re-track.
				rerun(false)
			},
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

// reactiveCleanups maps a node's reactive-id to its stop function so the
// per-node effect is released when the node leaves the DOM.
var reactiveCleanups = struct {
	mu     sync.Mutex
	store  map[string]func()
	nextID uint64
}{store: make(map[string]func())}

// registerReactiveCleanup associates a stop fn with a DOM text node.
// Text nodes can't hold attributes, so we stash an id as a JS property.
func registerReactiveCleanup(node js.Value, stop func()) {
	reactiveCleanups.mu.Lock()
	reactiveCleanups.nextID++
	id := reactiveCleanups.nextID
	reactiveCleanups.store[itoa(id)] = stop
	reactiveCleanups.mu.Unlock()
	node.Set("__rxReactiveId", itoa(id))
}

// releaseReactive runs and removes the cleanup for a node, if any.
// Called from releaseNode alongside event-listener cleanup.
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
