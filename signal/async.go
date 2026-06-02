// async.go — AsyncSignal for data fetching with loading/error/success states.
package signal

import "sync"

// State represents the lifecycle of an async operation
type State int

const (
	Loading State = iota
	Success
	Error
)

// AsyncSignal holds the result of an async operation.
// When state changes, notifies global listener (triggers re-render).
type AsyncSignal[T any] struct {
	mu    sync.RWMutex
	state State
	data  T
	err   error
	fn    func() (T, error)
}

// Async creates a new AsyncSignal and immediately starts fetching.
func Async[T any](fn func() (T, error)) *AsyncSignal[T] {
	guardRender("Async")
	a := &AsyncSignal[T]{state: Loading, fn: fn}
	go a.run()
	return a
}

func (a *AsyncSignal[T]) run() {
	// Notify loading state
	notifyGlobal()

	data, err := a.fn()

	a.mu.Lock()
	if err != nil {
		a.state = Error
		a.err = err
	} else {
		a.state = Success
		a.data = data
		a.err = nil
	}
	a.mu.Unlock()

	// Notify re-render on completion
	notifyGlobal()
}

// IsLoading returns true while fetching
func (a *AsyncSignal[T]) IsLoading() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.state == Loading
}

// IsError returns true if fetch failed
func (a *AsyncSignal[T]) IsError() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.state == Error
}

// IsSuccess returns true if data is ready
func (a *AsyncSignal[T]) IsSuccess() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.state == Success
}

// Data returns the fetched data (zero value if not loaded)
func (a *AsyncSignal[T]) Data() T {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.data
}

// Err returns the error (nil if no error)
func (a *AsyncSignal[T]) Err() error {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.err
}

// Refetch re-runs the async function
func (a *AsyncSignal[T]) Refetch() {
	a.mu.Lock()
	a.state = Loading
	a.mu.Unlock()
	notifyGlobal()
	go a.run()
}

// CurrentState returns the raw State value
func (a *AsyncSignal[T]) CurrentState() State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.state
}
