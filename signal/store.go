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
	mu   sync.RWMutex
	val  *T
	subs []*subscription
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

// Update applies a mutation function and notifies all subscribers.
// The mutation runs inside a lock — keep it fast, no I/O.
func (s *Store[T]) Update(fn func(*T)) {
	s.mu.Lock()
	fn(s.val)
	subs := make([]*subscription, len(s.subs))
	copy(subs, s.subs)
	s.mu.Unlock()

	for _, sub := range subs {
		sub.effect.run()
	}
	notifyGlobal()
}

// Set replaces the entire state value.
func (s *Store[T]) Set(val *T) {
	s.mu.Lock()
	s.val = val
	subs := make([]*subscription, len(s.subs))
	copy(subs, s.subs)
	s.mu.Unlock()

	for _, sub := range subs {
		sub.effect.run()
	}
	notifyGlobal()
}

// Reset replaces state with a fresh copy of initial.
func (s *Store[T]) Reset(initial *T) {
	s.Set(initial)
}
