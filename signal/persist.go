// persist.go — reactive signals backed by localStorage.
// rx.Persist("key", defaultValue) works exactly like rx.Use()
// but survives page refresh.
package signal

import "encoding/json"

// PersistSignal is a Signal that reads/writes localStorage automatically.
// The value is serialized as JSON.
//
// Usage:
//
//	// In Setup() — persists across page reloads
//	p.theme = rx.Persist("theme", "light")
//	p.token = rx.Persist("auth_token", "")
//
//	// Works exactly like a normal signal:
//	p.theme.Val()          // read (auto-tracked)
//	p.theme.Set("dark")    // write + saves to localStorage
type PersistSignal[T any] struct {
	Signal[T]
	key string
}

// NewPersist creates a signal backed by localStorage.
// storageWrite is injected by run_wasm.go (WASM) or is a no-op (native).
// storageRead is injected the same way.
var (
	StorageRead  func(key string) (string, bool)
	StorageWrite func(key, value string)
	StorageDel   func(key string)
)

// NewPersist creates a persistent signal.
// If a value exists in localStorage it overrides the default.
func NewPersist[T any](key string, defaultVal T) *PersistSignal[T] {
	val := defaultVal

	// Load from storage if available
	if StorageRead != nil {
		if raw, ok := StorageRead(key); ok && raw != "" {
			var stored T
			if err := json.Unmarshal([]byte(raw), &stored); err == nil {
				val = stored
			}
		}
	}

	p := &PersistSignal[T]{
		Signal: Signal[T]{val: val},
		key:    key,
	}
	return p
}

// Set updates the value, saves to localStorage, and triggers re-renders.
func (p *PersistSignal[T]) Set(v T) {
	// Save to localStorage
	if StorageWrite != nil {
		if b, err := json.Marshal(v); err == nil {
			StorageWrite(p.key, string(b))
		}
	}
	p.Signal.Set(v)
}

// Clear removes the value from localStorage and resets to zero value.
func (p *PersistSignal[T]) Clear() {
	if StorageDel != nil {
		StorageDel(p.key)
	}
	var zero T
	p.Signal.Set(zero)
}
