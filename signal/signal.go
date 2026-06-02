// Package signal is the reactive state engine of RyxoGo.
// Everything in the UI that changes lives here.
package signal

import "sync"

// ---------------------------------------------------------
// Tracker — auto-detects which signals a computed/effect reads
// ---------------------------------------------------------

var tracker struct {
	mu      sync.Mutex
	current *[]Reactive // currently active computed or effect
}

// Reactive is anything that can be subscribed to
type Reactive interface {
	subscribe(fn func())
}

// track records that the current computed/effect depends on r
func track(r Reactive) {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	if tracker.current != nil {
		*tracker.current = append(*tracker.current, r)
	}
}

// withTracking runs fn while recording all signals it reads
func withTracking(deps *[]Reactive, fn func()) {
	tracker.mu.Lock()
	prev := tracker.current
	tracker.current = deps
	tracker.mu.Unlock()

	fn()

	tracker.mu.Lock()
	tracker.current = prev
	tracker.mu.Unlock()
}

// ---------------------------------------------------------
// Signal[T] — a live variable. use() returns one of these.
// ---------------------------------------------------------

type Signal[T any] struct {
	mu        sync.RWMutex
	val       T
	listeners []func()
}

// use creates a new Signal with an initial value.
// This is the primary API — developers call use(0), use(""), use(false).
func New[T any](initial T) *Signal[T] {
	return &Signal[T]{val: initial}
}

// Val reads the current value.
// Automatically registers this signal as a dependency
// if called inside a computed() or watch().
func (s *Signal[T]) Val() T {
	track(s)
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.val
}

// Set updates the value and notifies all subscribers.
// This triggers re-renders for any component using this signal.
func (s *Signal[T]) Set(v T) {
	s.mu.Lock()
	s.val = v
	listeners := make([]func(), len(s.listeners))
	copy(listeners, s.listeners)
	s.mu.Unlock()

	// Notify global renderer first
	notifyGlobal()

	// Then notify specific subscribers (computed, effects)
	for _, fn := range listeners {
		fn()
	}
}

// Update applies a function to the current value — useful for slices/maps
func (s *Signal[T]) Update(fn func(T) T) {
	s.mu.RLock()
	current := s.val
	s.mu.RUnlock()
	s.Set(fn(current))
}

// subscribe registers a listener — called by effects and computed
func (s *Signal[T]) subscribe(fn func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listeners = append(s.listeners, fn)
}

// ---------------------------------------------------------
// Computed[T] — a derived signal. computed(() => a.Val() * 2)
// Auto-tracks dependencies, no array needed.
// ---------------------------------------------------------

type Computed[T any] struct {
	mu       sync.RWMutex
	val      T
	fn       func() T
	deps     []Reactive
	listeners []func()
}

// Derive creates a computed signal from a function.
// Automatically tracks which signals the function reads.
func Derive[T any](fn func() T) *Computed[T] {
	c := &Computed[T]{fn: fn}
	c.recompute()
	return c
}

func (c *Computed[T]) recompute() {
	var newDeps []Reactive
	var result T

	withTracking(&newDeps, func() {
		result = c.fn()
	})

	c.mu.Lock()
	c.val = result

	// subscribe to all newly detected dependencies
	for _, dep := range newDeps {
		dep.subscribe(func() {
			c.recompute()
			c.notify()
		})
	}
	c.deps = newDeps
	c.mu.Unlock()
}

// Val reads the computed value
func (c *Computed[T]) Val() T {
	track(c)
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.val
}

func (c *Computed[T]) subscribe(fn func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.listeners = append(c.listeners, fn)
}

func (c *Computed[T]) notify() {
	c.mu.RLock()
	listeners := make([]func(), len(c.listeners))
	copy(listeners, c.listeners)
	c.mu.RUnlock()
	for _, fn := range listeners {
		fn()
	}
}

// ---------------------------------------------------------
// Effect — runs a function when its dependencies change.
// watch(signal, func() { ... }) in the user API.
// ---------------------------------------------------------

type Effect struct {
	fn   func()
	deps []Reactive
}

// Watch creates an effect that re-runs when any read signal changes.
// No dependency array needed — RyxoGo tracks automatically.
func Watch(fn func()) *Effect {
	e := &Effect{fn: fn}
	e.run()
	return e
}

func (e *Effect) run() {
	var deps []Reactive
	withTracking(&deps, e.fn)

	// subscribe to all detected dependencies
	for _, dep := range deps {
		dep.subscribe(func() {
			e.fn()
		})
	}
	e.deps = deps
}

// ---------------------------------------------------------
// Global listener — wires all signals to the renderer
// ---------------------------------------------------------

var globalListener func()

// SetGlobalListener sets a function called whenever ANY signal changes.
// The renderer calls this before Setup() so signals auto-trigger re-renders.
// Pass nil to clear.
func SetGlobalListener(fn func()) {
	tracker.mu.Lock()
	globalListener = fn
	tracker.mu.Unlock()
}

// notifyGlobal calls the global listener if set
func notifyGlobal() {
	tracker.mu.Lock()
	fn := globalListener
	tracker.mu.Unlock()
	if fn != nil {
		fn()
	}
}
