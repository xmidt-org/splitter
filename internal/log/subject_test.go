// SPDX-FileCopyrightText: 2026 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package log

import (
	"bytes"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	logger := slog.New(handler)

	subject := New(logger)

	require.NotNil(t, subject)
	assert.Equal(t, 1, subject.Count(), "should have one listener attached (the default listener)")
}

func TestNew_SubjectReceivesEvents(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	logger := slog.New(handler)

	subject := New(logger)

	event := NewEvent(LevelInfo, "test message", map[string]any{"key": "value"})
	subject.NotifySync(event)

	// Give the logger time to write
	time.Sleep(10 * time.Millisecond)

	output := buf.String()
	assert.Contains(t, output, "test message")
	assert.Contains(t, output, "key")
	assert.Contains(t, output, "value")
}

func TestNew_MultipleEvents(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(handler)

	subject := New(logger)

	events := []Event{
		NewEvent(LevelDebug, "debug event", nil),
		NewEvent(LevelInfo, "info event", nil),
		NewEvent(LevelWarn, "warn event", nil),
		NewEvent(LevelError, "error event", nil),
	}

	for _, event := range events {
		subject.NotifySync(event)
	}

	time.Sleep(10 * time.Millisecond)

	output := buf.String()
	assert.Contains(t, output, "debug event")
	assert.Contains(t, output, "info event")
	assert.Contains(t, output, "warn event")
	assert.Contains(t, output, "error event")
}

func TestNew_AdditionalListeners(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	logger := slog.New(handler)

	subject := New(logger)

	var receivedEvents []Event
	var mu sync.Mutex

	// Add an additional listener
	detach := subject.Attach(func(event Event) {
		mu.Lock()
		defer mu.Unlock()
		receivedEvents = append(receivedEvents, event)
	})
	defer detach()

	assert.Equal(t, 2, subject.Count(), "should have two listeners")

	event := NewEvent(LevelInfo, "test", map[string]any{"data": 123})
	subject.NotifySync(event)

	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	require.Len(t, receivedEvents, 1)
	assert.Equal(t, "test", receivedEvents[0].Message)
	assert.Equal(t, 123, receivedEvents[0].Attrs["data"])

	// Verify slog also received it
	output := buf.String()
	assert.Contains(t, output, "test")
}

func TestNew_AsyncNotification(t *testing.T) {
	// Test async notification by verifying the attached observer is called in a goroutine
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	logger := slog.New(handler)

	subject := New(logger)

	var wg sync.WaitGroup
	wg.Add(1)

	observerCalled := false
	subject.Attach(func(event Event) {
		observerCalled = true
		wg.Done()
	})

	event := NewEvent(LevelInfo, "async test", nil)
	subject.Notify(event)

	// Wait for the async observer to be called
	wg.Wait()

	// Verify the observer was called
	assert.True(t, observerCalled)
}

func TestNew_ConcurrentNotifications(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	logger := slog.New(handler)

	subject := New(logger)

	var receivedCount int
	var mu sync.Mutex
	var wg sync.WaitGroup

	numEvents := 50
	wg.Add(numEvents)

	subject.Attach(func(event Event) {
		mu.Lock()
		defer mu.Unlock()
		receivedCount++
		wg.Done()
	})

	// Send events concurrently
	for i := 0; i < numEvents; i++ {
		go func(i int) {
			event := NewEvent(LevelInfo, "concurrent", map[string]any{"index": i})
			subject.NotifySync(event)
		}(i)
	}

	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, numEvents, receivedCount)
}

func TestNew_DetachListener(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	logger := slog.New(handler)

	subject := New(logger)

	var receivedCount int
	var mu sync.Mutex

	detach := subject.Attach(func(event Event) {
		mu.Lock()
		defer mu.Unlock()
		receivedCount++
	})

	initialCount := subject.Count()

	event1 := NewEvent(LevelInfo, "first", nil)
	subject.NotifySync(event1)

	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	count1 := receivedCount
	mu.Unlock()
	assert.Equal(t, 1, count1)

	// Detach the additional listener
	detach()
	assert.Equal(t, initialCount-1, subject.Count())

	event2 := NewEvent(LevelInfo, "second", nil)
	subject.NotifySync(event2)

	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	count2 := receivedCount
	mu.Unlock()
	assert.Equal(t, 1, count2, "detached listener should not receive events")
}

func TestNew_DifferentLogLevels(t *testing.T) {
	tests := []struct {
		name      string
		level     Level
		shouldLog bool
		minLevel  slog.Level
	}{
		{"debug allowed", LevelDebug, true, slog.LevelDebug},
		{"debug filtered", LevelDebug, false, slog.LevelInfo},
		{"info allowed", LevelInfo, true, slog.LevelInfo},
		{"warn allowed", LevelWarn, true, slog.LevelInfo},
		{"error allowed", LevelError, true, slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: tt.minLevel})
			logger := slog.New(handler)

			subject := New(logger)

			event := NewEvent(tt.level, "test message", nil)
			subject.NotifySync(event)

			time.Sleep(10 * time.Millisecond)

			output := buf.String()
			if tt.shouldLog {
				assert.Contains(t, output, "test message")
			} else {
				assert.Empty(t, output)
			}
		})
	}
}
