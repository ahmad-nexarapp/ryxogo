// types.go — makes signal types available directly as rx.Signal, rx.Computed etc.
// This lets developers write:
//
//	type Counter struct {
//	    rx.Page
//	    count *rx.Signal[int]   ← this works
//	}
//
// Go 1.22 doesn't support generic type aliases, so we re-export
// the signal package types by making them available via a blank import
// and documenting the correct usage pattern.
package ryxogo

import "github.com/ahmad-nexarapp/ryxogo/signal"

// Signal is a reactive variable. When its value changes,
// any component reading it automatically re-renders.
//
// Create with rx.Use():
//
//	count := rx.Use(0)         // returns *signal.Signal[int]
//	name  := rx.Use("")        // returns *signal.Signal[string]
//
// In your struct, declare the field type as *signal.Signal[T]:
//
//	import "github.com/ahmad-nexarapp/ryxogo/signal"
//
//	type Counter struct {
//	    rx.Page
//	    count *signal.Signal[int]
//	}
//
// Or use the shorthand package alias:
//
//	import sig "github.com/ahmad-nexarapp/ryxogo/signal"
//
//	type Counter struct {
//	    rx.Page
//	    count *sig.Signal[int]
//	}
var _ = signal.New[int] // keep import alive

// NewSignal creates a signal directly — same as Use() but explicit.
func NewSignal[T any](v T) *signal.Signal[T] {
	return signal.New(v)
}

// NewComputed creates a computed signal directly — same as Computed().
func NewComputed[T any](fn func() T) *signal.Computed[T] {
	return signal.Derive(fn)
}

// NewAsync creates an async signal directly — same as Async().
func NewAsync[T any](fn func() (T, error)) *signal.AsyncSignal[T] {
	return signal.Async(fn)
}
