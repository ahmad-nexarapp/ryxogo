// Package signal implements reactive state for RyxoGo.
// Uses automatic dependency tracking — no manual subscriptions needed.
//
// How it works:
//   During Render(), the renderer sets an active "effect" (the re-render fn).
//   Every signal.Val() call during Render() registers that signal as a dependency.
//   When any dependency changes via Set(), the effect re-runs (re-renders).
//
// This is the same model used by Solid.js, Vue 3, and Preact Signals.
package signal

import "sync"

// ---------------------------------------------------------
// Tracking context — which effect is currently running
// ---------------------------------------------------------

var tracking struct {
	mu     sync.Mutex
	active *effect // currently executing effect, nil if none
}

// effect represents a reactive computation (Render, Computed, Watch)
type effect struct {
	fn   func()        // the function to re-run
	deps []*subscriber // signals this effect depends on
}

// subscriber is the link between a signal and an effect
type subscriber struct {
	signal interface{ unsubscribe(*subscriber) }
	effect *effect
	fn     func()
}

// setActive sets the currently active effect and returns a cleanup function
func setActive(e *effect) func() {
	tracking.mu.Lock()
	prev := tracking.active
	tracking.active = e
	tracking.mu.Unlock()
	return func() {
		tracking.mu.Lock()
		tracking.active = prev
		tracking.mu.Unlock()
	}
}

// getActive returns the currently active effect
func getActive() *effect {
	tracking.mu.Lock()
	defer tracking.mu.Unlock()
	return tracking.active
}

// ---------------------------------------------------------
// Signal[T] — a reactive value
// ---------------------------------------------------------

// Signal holds a value and notifies dependents when it changes.
type Signal[T any] struct {
	mu   sync.RWMutex
	val  T
	subs []*subscriber
}

// New creates a Signal with an initial value.
// Developers use rx.Use() which calls this.
func New[T any](initial T) *Signal[T] {
	return &Signal[T]{val: initial}
}

// Val reads the current value AND registers the calling effect as a dependent.
// Call this inside Render() or Computed() — auto-tracked, no setup needed.
func (s *Signal[T]) Val() T {
	// If an effect is active, subscribe it to this signal
	if e := getActive(); e != nil {
		s.mu.Lock()
		// Check if already subscribed to avoid duplicates
		already := false
		for _, sub := range s.subs {
			if sub.effect == e {
				already = true
				break
			}
		}
		if !already {
			sub := &subscriber{signal: s, effect: e, fn: e.fn}
			s.subs = append(s.subs, sub)
			e.deps = append(e.deps, sub)
		}
		s.mu.Unlock()
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.val
}

// Set updates the value and re-runs all dependent effects.
func (s *Signal[T]) Set(v T) {
	s.mu.Lock()
	s.val = v
	subs := make([]*subscriber, len(s.subs))
	copy(subs, s.subs)
	s.mu.Unlock()

	for _, sub := range subs {
		sub.fn()
	}
}

// Update applies a transform function to the current value.
func (s *Signal[T]) Update(fn func(T) T) {
	s.mu.RLock()
	current := s.val
	s.mu.RUnlock()
	s.Set(fn(current))
}

func (s *Signal[T]) unsubscribe(sub *subscriber) {
	s.mu.Lock()
	defer s.mu.Unlock()
	filtered := s.subs[:0]
	for _, existing := range s.subs {
		if existing != sub {
			filtered = append(filtered, existing)
		}
	}
	s.subs = filtered
}

// ---------------------------------------------------------
// Computed[T] — a derived signal
// ---------------------------------------------------------

// Computed holds a value derived from other signals.
// Re-computes automatically when any dependency changes.
type Computed[T any] struct {
	mu   sync.RWMutex
	val  T
	fn   func() T
	subs []*subscriber
}

// Derive creates a Computed signal. Developers use rx.Computed().
func Derive[T any](fn func() T) *Computed[T] {
	c := &Computed[T]{fn: fn}
	c.recompute()
	return c
}

func (c *Computed[T]) recompute() {
	e := &effect{fn: func() { c.recompute(); c.notify() }}
	restore := setActive(e)
	val := c.fn()
	restore()

	c.mu.Lock()
	c.val = val
	c.mu.Unlock()
}

// Val reads the computed value and registers the calling effect as dependent.
func (c *Computed[T]) Val() T {
	if e := getActive(); e != nil {
		c.mu.Lock()
		already := false
		for _, sub := range c.subs {
			if sub.effect == e {
				already = true
				break
			}
		}
		if !already {
			sub := &subscriber{signal: c, effect: e, fn: e.fn}
			c.subs = append(c.subs, sub)
			e.deps = append(e.deps, sub)
		}
		c.mu.Unlock()
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.val
}

func (c *Computed[T]) notify() {
	c.mu.RLock()
	subs := make([]*subscriber, len(c.subs))
	copy(subs, c.subs)
	c.mu.RUnlock()
	for _, sub := range subs {
		sub.fn()
	}
}

func (c *Computed[T]) unsubscribe(sub *subscriber) {
	c.mu.Lock()
	defer c.mu.Unlock()
	filtered := c.subs[:0]
	for _, existing := range c.subs {
		if existing != sub {
			filtered = append(filtered, existing)
		}
	}
	c.subs = filtered
}

// ---------------------------------------------------------
// Effect — a side effect that re-runs when deps change
// ---------------------------------------------------------

// Effect runs a function and re-runs it when any signal it reads changes.
type Effect struct {
	e *effect
}

// Watch creates a reactive effect. Developers use rx.Watch().
func Watch(fn func()) *Effect {
	e := &effect{}
	e.fn = func() {
		// Clear old deps before re-running
		for _, dep := range e.deps {
			dep.signal.unsubscribe(dep)
		}
		e.deps = e.deps[:0]

		restore := setActive(e)
		fn()
		restore()
	}
	e.fn() // run immediately to collect deps
	return &Effect{e: e}
}

// Stop removes all subscriptions — call when component unmounts
func (ef *Effect) Stop() {
	for _, dep := range ef.e.deps {
		dep.signal.unsubscribe(dep)
	}
	ef.e.deps = nil
}

// ---------------------------------------------------------
// RunTracked — run a function while tracking signal reads
// Used by the renderer to track which signals Render() reads
// ---------------------------------------------------------

// RunTracked runs fn while tracking signal reads.
// Returns a cleanup function that removes all subscriptions.
// When any tracked signal changes, fn is re-run and dependencies re-collected.
// This handles conditional reads correctly (deps change each render).
func RunTracked(fn func(), onChange func()) func() {
	e := &effect{}

	rerun := func() {
		// Clear ALL old subscriptions before re-running
		// This prevents double-fire and stale deps
		for _, dep := range e.deps {
			dep.signal.unsubscribe(dep)
		}
		e.deps = e.deps[:0]

		// Re-run fn, collecting fresh deps
		restore := setActive(e)
		fn()
		restore()

		// Notify caller that a re-render happened
		onChange()
	}

	e.fn = rerun

	// Initial run — just collect deps, no onChange call
	restore := setActive(e)
	fn()
	restore()

	// Return cleanup
	return func() {
		for _, dep := range e.deps {
			dep.signal.unsubscribe(dep)
		}
		e.deps = nil
	}
}

// ---------------------------------------------------------
// Global listener (kept for AsyncSignal compat)
// ---------------------------------------------------------

var globalMu sync.Mutex
var globalListener func()

// SetGlobalListener sets a fallback called when any signal changes.
// Used by AsyncSignal which can't be tracked during Render().
func SetGlobalListener(fn func()) {
	globalMu.Lock()
	globalListener = fn
	globalMu.Unlock()
}

func notifyGlobal() {
	globalMu.Lock()
	fn := globalListener
	globalMu.Unlock()
	if fn != nil {
		fn()
	}
}

// Track runs fn while tracking signal reads.
// When a tracked signal changes, onChange is called exactly once.
// Returns a stop function that removes all subscriptions.
//
// This is the clean version of RunTracked:
// - fn runs once to collect deps
// - When any dep changes, onChange fires (once, not fn again)
// - Caller decides what to do in onChange (e.g. schedule rAF + re-call Track)
func Track(fn func(), onChange func()) func() {
	e := &effect{fn: onChange}

	restore := setActive(e)
	fn()
	restore()

	return func() {
		for _, dep := range e.deps {
			dep.signal.unsubscribe(dep)
		}
		e.deps = nil
	}
}
