// SPDX-FileCopyrightText: 2026 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

// Package observe provides an implementation of the observer pattern with generic event types.
package observe

import (
	"sync"
)

// Observer is a function that receives events of type T.
type Observer[T any] func(T)

// observerEntry wraps an observer with a detached flag.
type observerEntry[T any] struct {
	observer Observer[T]
	detached *bool
}

// Subject manages observers and notifies them of events.
type Subject[T any] struct {
	observers []observerEntry[T]
	mu        sync.RWMutex
}

// NewSubject creates a new Subject for events of type T.
func NewSubject[T any]() *Subject[T] {
	return &Subject[T]{
		observers: make([]observerEntry[T], 0),
	}
}

// Attach registers an observer to receive events.
// Returns a function that can be called to detach the observer.
func (s *Subject[T]) Attach(observer Observer[T]) func() {
	s.mu.Lock()
	defer s.mu.Unlock()

	detached := false
	entry := observerEntry[T]{
		observer: observer,
		detached: &detached,
	}
	s.observers = append(s.observers, entry)

	// Return detach function that sets the flag and compacts
	return func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		*entry.detached = true
		s.compact()
	}
}

// compact removes all detached observers.
// Must be called with lock held.
func (s *Subject[T]) compact() {
	compacted := make([]observerEntry[T], 0, len(s.observers))
	for _, entry := range s.observers {
		if !*entry.detached {
			compacted = append(compacted, entry)
		}
	}
	s.observers = compacted
}

// NotifySync sends an event to all registered observers synchronously.
func (s *Subject[T]) NotifySync(event T) {
	s.mu.RLock()
	observers := make([]Observer[T], 0, len(s.observers))
	for _, entry := range s.observers {
		if !*entry.detached {
			observers = append(observers, entry.observer)
		}
	}
	s.mu.RUnlock()

	for _, observer := range observers {
		observer(event)
	}
}

// Notify sends an event to all registered observers asynchronously.
// Each observer is called in its own goroutine to prevent blocking.
func (s *Subject[T]) Notify(event T) {
	s.mu.RLock()
	observers := make([]Observer[T], 0, len(s.observers))
	for _, entry := range s.observers {
		if !*entry.detached {
			observers = append(observers, entry.observer)
		}
	}
	s.mu.RUnlock()

	for _, observer := range observers {
		go observer(event)
	}
}

// Count returns the number of active observers.
func (s *Subject[T]) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, entry := range s.observers {
		if !*entry.detached {
			count++
		}
	}
	return count
}

// Clear removes all observers.
func (s *Subject[T]) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.observers = make([]observerEntry[T], 0)
}
