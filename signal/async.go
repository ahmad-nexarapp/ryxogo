// async.go — AsyncSignal powers fetch() in RyxoGo.
// One call replaces useState + useEffect + loading + error boilerplate.
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
// Developers check .Loading(), .Data(), .Err() — never manage state manually.
type AsyncSignal[T any] struct {
	mu        sync.RWMutex
	state     State
	data      T
	err       error
	fn        func() (T, error)
	listeners []func()
}

// Async creates a new AsyncSignal and immediately starts fetching.
// fetch("/api/users") returns one of these.
func Async[T any](fn func() (T, error)) *AsyncSignal[T] {
	a := &AsyncSignal[T]{
		state: Loading,
		fn:    fn,
	}
	go a.run()
	return a
}

func (a *AsyncSignal[T]) run() {
	a.setState(Loading)
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
	listeners := make([]func(), len(a.listeners))
	copy(listeners, a.listeners)
	a.mu.Unlock()

	// notify all subscribers (triggers re-render)
	notifyGlobal()
	for _, fn := range listeners {
		fn()
	}
}

func (a *AsyncSignal[T]) setState(s State) {
	a.mu.Lock()
	a.state = s
	listeners := make([]func(), len(a.listeners))
	copy(listeners, a.listeners)
	a.mu.Unlock()

	for _, fn := range listeners {
		fn()
	}
}

// IsLoading returns true while the async operation is in progress
func (a *AsyncSignal[T]) IsLoading() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.state == Loading
}

// IsError returns true if the async operation failed
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

// Data returns the result — nil if not yet loaded
func (a *AsyncSignal[T]) Data() T {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.data
}

// Err returns the error — nil if no error
func (a *AsyncSignal[T]) Err() error {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.err
}

// Refetch re-runs the async function from scratch
func (a *AsyncSignal[T]) Refetch() {
	go a.run()
}

// CurrentState returns the raw State value for switch statements
func (a *AsyncSignal[T]) CurrentState() State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.state
}

// subscribe allows components to re-render when this signal changes
func (a *AsyncSignal[T]) subscribe(fn func()) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.listeners = append(a.listeners, fn)
}
