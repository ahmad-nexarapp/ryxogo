//go:build wasm

// funcstore_wasm.go — tracks js.Func values so they can be Released.
// js.FuncOf allocates a Go→JS bridge that is never GC'd unless Release() is called.
// We generate a unique ID per element and store funcs by that ID.
package renderer

import (
	"fmt"
	"sync"
	"sync/atomic"
	"syscall/js"
)

var nextID uint64

// funcStore holds js.Func values keyed by element ID string
type funcStore struct {
	mu    sync.Mutex
	store map[string][]js.Func
}

var funcs = &funcStore{
	store: make(map[string][]js.Func),
}

// getOrSetID returns the ryxo-id attribute of el, creating it if absent
func getOrSetID(el js.Value) string {
	existing := el.Call("getAttribute", "data-ryxo-id")
	if !existing.IsNull() && !existing.IsUndefined() && existing.String() != "" {
		return existing.String()
	}
	id := fmt.Sprintf("r%d", atomic.AddUint64(&nextID, 1))
	el.Call("setAttribute", "data-ryxo-id", id)
	return id
}

// makeFunc creates a js.Func, registers it for later cleanup, and returns it.
func (r *Renderer) makeFunc(el js.Value, fn func(js.Value, []js.Value) interface{}) js.Func {
	f := js.FuncOf(fn)
	id := getOrSetID(el)
	funcs.mu.Lock()
	funcs.store[id] = append(funcs.store[id], f)
	funcs.mu.Unlock()
	return f
}

// releaseNode frees event listener funcs AND reactive text effects for a node.
// Called before removeChild or replaceChild. Works for both element nodes
// (which carry listeners via data-ryxo-id) and text nodes (which carry
// reactive effects via the __rxReactiveId JS property).
func releaseNode(ref interface{}) {
	if ref == nil {
		return
	}
	el, ok := ref.(js.Value)
	if !ok {
		return
	}

	// Release any fine-grained reactive text effect on this node.
	releaseReactive(ref)

	// Element nodes also need their event-listener funcs released.
	// nodeType 1 == element; text nodes (3) have no listeners and no
	// getAttribute, so we skip them.
	if el.Get("nodeType").Int() != 1 {
		return
	}
	idVal := el.Call("getAttribute", "data-ryxo-id")
	if idVal.IsNull() || idVal.IsUndefined() {
		return
	}
	id := idVal.String()
	if id == "" {
		return
	}
	funcs.mu.Lock()
	fns := funcs.store[id]
	delete(funcs.store, id)
	funcs.mu.Unlock()

	for _, f := range fns {
		f.Release()
	}
}
