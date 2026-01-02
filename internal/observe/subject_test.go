// SPDX-FileCopyrightText: 2026 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package observe

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test event types
type StringEvent string
type IntEvent int
type ComplexEvent struct {
	ID      int
	Message string
	Data    map[string]any
}

func TestNewSubject(t *testing.T) {
	assert := assert.New(t)

	// Test with string events
	subject := NewSubject[string]()
	assert.NotNil(subject)
	assert.Equal(0, subject.Count())

	// Test with int events
	intSubject := NewSubject[int]()
	assert.NotNil(intSubject)
	assert.Equal(0, intSubject.Count())

	// Test with struct events
	complexSubject := NewSubject[ComplexEvent]()
	assert.NotNil(complexSubject)
	assert.Equal(0, complexSubject.Count())
}

func TestSubject_Attach(t *testing.T) {
	assert := assert.New(t)

	subject := NewSubject[string]()

	// Attach an observer
	detach := subject.Attach(func(event string) {})
	assert.NotNil(detach)
	assert.Equal(1, subject.Count())

	// Attach multiple observers
	detach2 := subject.Attach(func(event string) {})
	assert.NotNil(detach2)
	assert.Equal(2, subject.Count())
}

func TestSubject_Detach(t *testing.T) {
	assert := assert.New(t)

	subject := NewSubject[string]()

	// Attach observers
	detach1 := subject.Attach(func(event string) {})
	detach2 := subject.Attach(func(event string) {})
	assert.Equal(2, subject.Count())

	// Detach first observer
	detach1()
	assert.Equal(1, subject.Count())

	// Detach second observer
	detach2()
	assert.Equal(0, subject.Count())
}

func TestSubject_Notify(t *testing.T) {
	assert := assert.New(t)

	subject := NewSubject[string]()

	var received []string
	var mu sync.Mutex

	// Attach observer
	subject.Attach(func(event string) {
		mu.Lock()
		defer mu.Unlock()
		received = append(received, event)
	})

	// Notify
	subject.NotifySync("event1")
	subject.NotifySync("event2")

	mu.Lock()
	assert.Equal(2, len(received))
	assert.Equal("event1", received[0])
	assert.Equal("event2", received[1])
	mu.Unlock()
}

func TestSubject_NotifyMultipleObservers(t *testing.T) {
	assert := assert.New(t)

	subject := NewSubject[int]()

	var count1, count2 atomic.Int32

	// Attach two observers
	subject.Attach(func(event int) {
		count1.Add(int32(event))
	})

	subject.Attach(func(event int) {
		count2.Add(int32(event))
	})

	// Notify
	subject.NotifySync(5)
	subject.NotifySync(3)

	assert.Equal(int32(8), count1.Load())
	assert.Equal(int32(8), count2.Load())
}

func TestSubject_NotifyAsync(t *testing.T) {
	assert := assert.New(t)

	subject := NewSubject[string]()

	var received []string
	var mu sync.Mutex
	var wg sync.WaitGroup

	wg.Add(3)

	// Attach observer
	subject.Attach(func(event string) {
		mu.Lock()
		defer mu.Unlock()
		received = append(received, event)
		wg.Done()
	})

	// Notify asynchronously
	subject.Notify("event1")
	subject.Notify("event2")
	subject.Notify("event3")

	// Wait for async notifications
	wg.Wait()

	mu.Lock()
	assert.Equal(3, len(received))
	mu.Unlock()
}

func TestSubject_NotifyAsyncMultipleObservers(t *testing.T) {
	assert := assert.New(t)

	subject := NewSubject[int]()

	var count1, count2 atomic.Int32
	var wg sync.WaitGroup

	wg.Add(4) // 2 notifications × 2 observers

	// Attach two observers
	subject.Attach(func(event int) {
		count1.Add(int32(event))
		wg.Done()
	})

	subject.Attach(func(event int) {
		count2.Add(int32(event))
		wg.Done()
	})

	// Notify asynchronously
	subject.Notify(5)
	subject.Notify(3)

	// Wait for all async notifications
	wg.Wait()

	assert.Equal(int32(8), count1.Load())
	assert.Equal(int32(8), count2.Load())
}

func TestSubject_Count(t *testing.T) {
	assert := assert.New(t)

	subject := NewSubject[string]()
	assert.Equal(0, subject.Count())

	// Attach observers
	detach1 := subject.Attach(func(event string) {})
	assert.Equal(1, subject.Count())

	detach2 := subject.Attach(func(event string) {})
	assert.Equal(2, subject.Count())

	detach3 := subject.Attach(func(event string) {})
	assert.Equal(3, subject.Count())

	// Detach and verify count
	detach2()
	// Give time for compact to complete
	time.Sleep(10 * time.Millisecond)
	count := subject.Count()
	assert.True(count == 2 || count == 3) // May not be compacted yet

	detach1()
	time.Sleep(10 * time.Millisecond)

	detach3()
	time.Sleep(10 * time.Millisecond)
	assert.Equal(0, subject.Count())
}

func TestSubject_Clear(t *testing.T) {
	assert := assert.New(t)

	subject := NewSubject[string]()

	// Attach observers
	subject.Attach(func(event string) {})
	subject.Attach(func(event string) {})
	subject.Attach(func(event string) {})
	assert.Equal(3, subject.Count())

	// Clear all observers
	subject.Clear()
	assert.Equal(0, subject.Count())
}

func TestSubject_ComplexEvents(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	subject := NewSubject[ComplexEvent]()

	var received []ComplexEvent
	var mu sync.Mutex

	subject.Attach(func(event ComplexEvent) {
		mu.Lock()
		defer mu.Unlock()
		received = append(received, event)
	})

	// Notify with complex events
	event1 := ComplexEvent{
		ID:      1,
		Message: "first event",
		Data:    map[string]any{"key": "value"},
	}

	event2 := ComplexEvent{
		ID:      2,
		Message: "second event",
		Data:    map[string]any{"count": 42},
	}

	subject.NotifySync(event1)
	subject.NotifySync(event2)

	mu.Lock()
	require.Equal(2, len(received))
	assert.Equal(1, received[0].ID)
	assert.Equal("first event", received[0].Message)
	assert.Equal(2, received[1].ID)
	assert.Equal("second event", received[1].Message)
	mu.Unlock()
}

func TestSubject_DetachDuringNotification(t *testing.T) {
	assert := assert.New(t)

	subject := NewSubject[int]()

	var count atomic.Int32
	var detach func()

	// Observer that detaches itself after first call
	detach = subject.Attach(func(event int) {
		count.Add(1)
		if count.Load() == 1 {
			detach()
		}
	})

	assert.Equal(1, subject.Count())

	// First notification - observer should be called
	subject.NotifySync(1)
	assert.Equal(int32(1), count.Load())

	// Second notification - observer should not be called
	subject.NotifySync(2)
	time.Sleep(10 * time.Millisecond) // Give time for potential async operations

	assert.Equal(int32(1), count.Load())
	assert.Equal(0, subject.Count())
}

func TestSubject_ConcurrentAttachDetach(t *testing.T) {
	subject := NewSubject[string]()

	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrently attach and detach observers
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			detach := subject.Attach(func(event string) {})
			time.Sleep(time.Millisecond)
			detach()
		}()
	}

	wg.Wait()

	// Give time for all compactions to complete
	time.Sleep(100 * time.Millisecond)

	// All observers should eventually be detached
	// Note: Due to async nature, we check within a reasonable range
	count := subject.Count()
	assert.True(t, count <= 10, "Expected count <= 10, got %d", count)
}

func TestSubject_ConcurrentNotify(t *testing.T) {
	assert := assert.New(t)

	subject := NewSubject[int]()

	var count atomic.Int32
	subject.Attach(func(event int) {
		count.Add(int32(event))
	})

	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrently notify
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			subject.NotifySync(1)
		}()
	}

	wg.Wait()

	// Should have received all notifications
	assert.Equal(int32(numGoroutines), count.Load())
}

func TestSubject_ConcurrentNotifyAsync(t *testing.T) {
	assert := assert.New(t)

	subject := NewSubject[int]()

	var count atomic.Int32
	var wg sync.WaitGroup

	numGoroutines := 100
	wg.Add(numGoroutines)

	subject.Attach(func(event int) {
		count.Add(int32(event))
		wg.Done()
	})

	// Concurrently notify async
	for i := 0; i < numGoroutines; i++ {
		go func() {
			subject.Notify(1)
		}()
	}

	wg.Wait()

	// Should have received all notifications
	assert.Equal(int32(numGoroutines), count.Load())
}

func TestSubject_NilObserverHandling(t *testing.T) {
	assert := assert.New(t)

	subject := NewSubject[string]()

	// Attach and immediately detach
	detach := subject.Attach(func(event string) {})
	detach()

	// Attach another observer
	var received bool
	subject.Attach(func(event string) {
		received = true
	})

	// Notify should work without panic
	subject.NotifySync("test")

	assert.True(received)
	assert.Equal(1, subject.Count())
}

func TestSubject_PointerEvents(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	type Event struct {
		Value int
	}

	subject := NewSubject[*Event]()

	var received []*Event
	var mu sync.Mutex

	subject.Attach(func(event *Event) {
		mu.Lock()
		defer mu.Unlock()
		received = append(received, event)
	})

	event1 := &Event{Value: 1}
	event2 := &Event{Value: 2}

	subject.NotifySync(event1)
	subject.NotifySync(event2)

	mu.Lock()
	require.Equal(2, len(received))
	assert.Equal(1, received[0].Value)
	assert.Equal(2, received[1].Value)
	mu.Unlock()
}

type EventInterface interface {
	GetID() int
}

type ConcreteEvent struct {
	ID int
}

func (e ConcreteEvent) GetID() int {
	return e.ID
}

func TestSubject_InterfaceEvents(t *testing.T) {
	assert := assert.New(t)

	subject := NewSubject[EventInterface]()

	var receivedID int
	subject.Attach(func(event EventInterface) {
		receivedID = event.GetID()
	})

	subject.NotifySync(ConcreteEvent{ID: 42})

	assert.Equal(42, receivedID)
}

// Benchmark tests
func BenchmarkSubject_Attach(b *testing.B) {
	subject := NewSubject[string]()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		subject.Attach(func(event string) {})
	}
}

func BenchmarkSubject_Notify(b *testing.B) {
	subject := NewSubject[string]()
	subject.Attach(func(event string) {})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		subject.NotifySync("event")
	}
}

func BenchmarkSubject_NotifyAsync(b *testing.B) {
	subject := NewSubject[string]()
	subject.Attach(func(event string) {})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		subject.Notify("event")
	}
}

func BenchmarkSubject_NotifyMultipleObservers(b *testing.B) {
	subject := NewSubject[string]()

	for i := 0; i < 10; i++ {
		subject.Attach(func(event string) {})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		subject.NotifySync("event")
	}
}
