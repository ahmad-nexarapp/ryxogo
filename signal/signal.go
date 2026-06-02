// Package signal implements reactive state for RyxoGo.
// Uses automatic dependency tracking — identical to Solid.js / Vue 3 Signals.
package signal

import "sync"

// ---------------------------------------------------------
// Tracking context
// ---------------------------------------------------------

var tracking struct {
	mu     sync.Mutex
	active *Effect
}

func setActive(e *Effect) func() {
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

func getActive() *Effect {
	tracking.mu.Lock()
	defer tracking.mu.Unlock()
	return tracking.active
}

// ---------------------------------------------------------
// subscription — link between a Signal and an Effect
// ---------------------------------------------------------

type subscription struct {
	cleanup func() // removes this sub from the signal's list
	effect  *Effect
}

// ---------------------------------------------------------
// Signal[T]
// ---------------------------------------------------------

type Signal[T any] struct {
	mu   sync.RWMutex
	val  T
	subs []*subscription
}

func New[T any](v T) *Signal[T] { return &Signal[T]{val: v} }

// Val reads the value and auto-subscribes the active Effect.
func (s *Signal[T]) Val() T {
	if e := getActive(); e != nil {
		e.track(func(sub *subscription) {
			s.mu.Lock()
			s.subs = append(s.subs, sub)
			s.mu.Unlock()
			sub.cleanup = func() {
				s.mu.Lock()
				for i, x := range s.subs {
					if x == sub {
						s.subs = append(s.subs[:i], s.subs[i+1:]...)
						break
					}
				}
				s.mu.Unlock()
			}
		})
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.val
}

// Set updates the value and re-runs all dependent Effects.
func (s *Signal[T]) Set(v T) {
	s.mu.Lock()
	s.val = v
	subs := make([]*subscription, len(s.subs))
	copy(subs, s.subs)
	s.mu.Unlock()

	for _, sub := range subs {
		sub.effect.run()
	}
	notifyGlobal()
}

// Update applies fn to the current value.
func (s *Signal[T]) Update(fn func(T) T) {
	s.mu.RLock()
	v := s.val
	s.mu.RUnlock()
	s.Set(fn(v))
}

// ---------------------------------------------------------
// Computed[T] — F1 FIX: clears old subs before each recompute
// ---------------------------------------------------------

type Computed[T any] struct {
	mu   sync.RWMutex
	val  T
	fn   func() T
	subs []*subscription
	eff  *Effect
}

func Derive[T any](fn func() T) *Computed[T] {
	c := &Computed[T]{fn: fn}
	// Create a single Effect that re-runs when any dep changes
	c.eff = newEffect(func() {
		c.mu.Lock()
		c.val = fn()
		subs := make([]*subscription, len(c.subs))
		copy(subs, c.subs)
		c.mu.Unlock()
		// Notify downstream dependents
		for _, sub := range subs {
			sub.effect.run()
		}
		notifyGlobal()
	})
	c.eff.run()
	return c
}

func (c *Computed[T]) Val() T {
	if e := getActive(); e != nil {
		e.track(func(sub *subscription) {
			c.mu.Lock()
			c.subs = append(c.subs, sub)
			c.mu.Unlock()
			sub.cleanup = func() {
				c.mu.Lock()
				for i, x := range c.subs {
					if x == sub {
						c.subs = append(c.subs[:i], c.subs[i+1:]...)
						break
					}
				}
				c.mu.Unlock()
			}
		})
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.val
}

// Stop removes all subscriptions — call when component unmounts
func (c *Computed[T]) Stop() { c.eff.cleanup() }

// ---------------------------------------------------------
// Effect — reactive side effect
// ---------------------------------------------------------

// Effect tracks signal dependencies and re-runs when they change.
type Effect struct {
	fn        func()          // the function to run
	subs      []*subscription // current subscriptions (cleared each run)
	mu        sync.Mutex
	cancelled bool            // set by cleanup() — prevents stale fire
}

func newEffect(fn func()) *Effect {
	return &Effect{fn: fn}
}

// run executes the effect, clearing old deps first (F1 FIX).
func (e *Effect) run() {
	e.mu.Lock()
	if e.cancelled {
		e.mu.Unlock()
		return
	}
	e.mu.Unlock()

	e.cleanup() // clear ALL old subscriptions before re-running
	restore := setActive(e)
	e.fn()
	restore()
}

// cleanup removes all subscriptions — called before each run and on Stop()
func (e *Effect) cleanup() {
	e.mu.Lock()
	subs := e.subs
	e.subs = nil
	e.mu.Unlock()
	for _, sub := range subs {
		if sub.cleanup != nil {
			sub.cleanup()
		}
	}
}

// Stop removes all subscriptions permanently and cancels future runs
func (e *Effect) Stop() {
	e.mu.Lock()
	e.cancelled = true
	e.mu.Unlock()
	e.cleanup()
}

// track registers a new subscription for this effect
func (e *Effect) track(register func(*subscription)) {
	sub := &subscription{effect: e}
	register(sub)
	e.mu.Lock()
	e.subs = append(e.subs, sub)
	e.mu.Unlock()
}

// ---------------------------------------------------------
// Watch — public side effect API (rx.Watch)
// ---------------------------------------------------------

func Watch(fn func()) *Effect {
	e := newEffect(fn)
	e.run()
	return e
}

// ---------------------------------------------------------
// Track tracks signal reads inside fn, calls onChange when any dep changes.
// Returns a stop function — call it before the next render to prevent
// stale effect calls when signals change during the cleanup window.
func Track(fn func(), onChange func()) func() {
	e := newEffect(onChange)
	// Run fn under tracking to collect deps — don't call onChange yet
	restore := setActive(e)
	fn()
	restore()
	return e.Stop // Stop cancels the effect AND removes all subscriptions
}

// ---------------------------------------------------------
// Global listener — for AsyncSignal
// ---------------------------------------------------------

var globalMu sync.Mutex
var globalListener func()

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
