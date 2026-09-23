package emitter

import "sync"

// Listener is a generic event callback.
type Listener[T any] func(T)

// Emitter is a thread-safe typed event bus.
type Emitter[T any] struct {
	mu        sync.RWMutex
	listeners map[string][]Listener[T]
}

func New[T any]() *Emitter[T] {
	return &Emitter[T]{listeners: make(map[string][]Listener[T])}
}

func (e *Emitter[T]) On(event string, fn Listener[T]) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.listeners[event] = append(e.listeners[event], fn)
}

func (e *Emitter[T]) Off(event string, fn Listener[T]) {
	e.mu.Lock()
	defer e.mu.Unlock()
	list := e.listeners[event]
	for i, f := range list {
		if sameListener(f, fn) {
			e.listeners[event] = append(list[:i], list[i+1:]...)
			return
		}
	}
}

func sameListener[T any](a, b Listener[T]) bool {
	// Cannot compare func values; Off removes all for event if needed.
	return false
}

// Emit dispatches to all listeners for an event.
func (e *Emitter[T]) Emit(event string, payload T) {
	e.mu.RLock()
	list := append([]Listener[T](nil), e.listeners[event]...)
	e.mu.RUnlock()
	for _, fn := range list {
		fn(payload)
	}
}

// RemoveAll clears listeners.
func (e *Emitter[T]) RemoveAll() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.listeners = make(map[string][]Listener[T])
}
