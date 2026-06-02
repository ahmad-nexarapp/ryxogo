// store.go — global reactive state store for RyxoGo.
// rx.NewStore, rx.GetStore, rx.UpdateStore are all implemented here.
package signal

import "sync"

// Store holds shared state accessible from any component.
// When the state changes, all components reading it re-render.
//
// Usage:
//
//	var Auth = rx.NewStore(&AuthState{})
//
//	type AuthState struct {
//	    User  *User
//	    Token string
//	}
//
//	// Read in any component:
//	auth := rx.GetStore(Auth)
//	auth.User.Name
//
//	// Write (triggers re-renders):
//	rx.UpdateStore(Auth, func(s *AuthState) {
//	    s.User = loggedInUser
//	    s.Token = res.Token
//	})
type Store[T any] struct {
	mu        sync.RWMutex
	val       *T
	subs      []*subscription
	selectors []*selectorSub // field-scoped subscriptions
}

// NewStore creates a new store with an initial value.
func NewStore[T any](initial *T) *Store[T] {
	return &Store[T]{val: initial}
}

// Get returns the current state and auto-subscribes the active Effect.
// Call inside Render() or Computed() for reactive reads.
func (s *Store[T]) Get() *T {
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

// Update applies a mutation function and notifies subscribers.
// Whole-store Get() subscribers always re-run; Select() subscribers re-run
// only if their selected slice actually changed.
func (s *Store[T]) Update(fn func(*T)) {
	s.mu.Lock()
	fn(s.val)
	subs := make([]*subscription, len(s.subs))
	copy(subs, s.subs)
	sels := make([]*selectorSub, len(s.selectors))
	copy(sels, s.selectors)
	s.mu.Unlock()

	for _, sub := range subs {
		sub.effect.run()
	}
	s.fireChangedSelectors(sels)
	notifyGlobal()
}

// Set replaces the entire state value.
func (s *Store[T]) Set(val *T) {
	s.mu.Lock()
	s.val = val
	subs := make([]*subscription, len(s.subs))
	copy(subs, s.subs)
	sels := make([]*selectorSub, len(s.selectors))
	copy(sels, s.selectors)
	s.mu.Unlock()

	for _, sub := range subs {
		sub.effect.run()
	}
	s.fireChangedSelectors(sels)
	notifyGlobal()
}

// fireChangedSelectors re-runs only selector subscriptions whose value changed.
func (s *Store[T]) fireChangedSelectors(sels []*selectorSub) {
	for _, ss := range sels {
		if ss.sel == nil {
			continue
		}
		now := ss.sel()
		if now != ss.last {
			ss.last = now
			if ss.sub != nil && ss.sub.effect != nil {
				ss.sub.effect.run()
			}
		}
	}
}

// Reset replaces state with a fresh copy of initial.
func (s *Store[T]) Reset(initial *T) {
	s.Set(initial)
}

// ---------------------------------------------------------
// Scoped subscriptions — only re-render when a SELECTED slice changes
// ---------------------------------------------------------

// selectorSub is a subscription bound to a selector's last value.
type selectorSub struct {
	sub  *subscription
	last interface{}
	sel  func() interface{}
}

// Select reads a derived slice of the store and subscribes the active effect
// ONLY to that slice. When Update runs, a selector subscriber re-renders only
// if its selected value actually changed (compared with ==).
//
// This solves the "everything re-renders" problem: a layout that selects
// SidebarMobileOpen won't re-render when an unrelated field like User changes.
//
//	open := rx.Select(UIStore, func(s *UIState) any { return s.SidebarMobileOpen })
//	// re-renders only when SidebarMobileOpen flips, not on every store update
//
// The selected value must be comparable with == (bool, string, number,
// pointer, small struct). For slices/maps, select a length or a version
// counter instead.
func Select[T any](s *Store[T], selector func(*T) interface{}) interface{} {
	cur := func() interface{} {
		s.mu.RLock()
		defer s.mu.RUnlock()
		return selector(s.val)
	}()

	if e := getActive(); e != nil {
		ss := &selectorSub{last: cur}
		ss.sel = func() interface{} {
			s.mu.RLock()
			defer s.mu.RUnlock()
			return selector(s.val)
		}
		e.track(func(sub *subscription) {
			ss.sub = sub
			s.mu.Lock()
			s.selectors = append(s.selectors, ss)
			s.mu.Unlock()
			sub.cleanup = func() {
				s.mu.Lock()
				for i, x := range s.selectors {
					if x == ss {
						s.selectors = append(s.selectors[:i], s.selectors[i+1:]...)
						break
					}
				}
				s.mu.Unlock()
			}
		})
	}
	return cur
}
