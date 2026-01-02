// SPDX-FileCopyrightText: 2026 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package log

import (
	"bytes"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewObserver(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	logger := slog.New(handler)

	observer := NewObserver(logger)

	require.NotNil(t, observer)
	require.NotNil(t, observer.logger)
}

func TestObserver_HandleEvent_Debug(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(handler)
	Observer := NewObserver(logger)

	event := NewEvent(LevelDebug, "debug message", map[string]any{"key": "value"})
	Observer.HandleEvent(event)

	output := buf.String()
	assert.Contains(t, output, "debug message")
	assert.Contains(t, output, "DEBUG")
}

func TestObserver_HandleEvent_Info(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	logger := slog.New(handler)
	Observer := NewObserver(logger)

	event := NewEvent(LevelInfo, "info message", map[string]any{"user": "alice"})
	Observer.HandleEvent(event)

	output := buf.String()
	assert.Contains(t, output, "info message")
	assert.Contains(t, output, "alice")
}

func TestObserver_HandleEvent_Warn(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	logger := slog.New(handler)
	observer := NewObserver(logger)

	event := NewEvent(LevelWarn, "warning message", nil)
	observer.HandleEvent(event)

	output := buf.String()
	assert.Contains(t, output, "warning message")
	assert.Contains(t, output, "WARN")
}

func TestObserver_HandleEvent_Error(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	logger := slog.New(handler)
	observer := NewObserver(logger)

	event := NewEvent(LevelError, "error occurred", map[string]any{"code": 500})
	observer.HandleEvent(event)

	output := buf.String()
	assert.Contains(t, output, "error occurred")
	assert.Contains(t, output, "ERROR")
	assert.Contains(t, output, "500")
}

func TestObserver_HandleEvent_WithAttributes(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	logger := slog.New(handler)
	observer := NewObserver(logger)
	attrs := map[string]any{
		"user":    "alice",
		"action":  "login",
		"success": true,
		"count":   42,
	}
	event := NewEvent(LevelInfo, "user action", attrs)
	observer.HandleEvent(event)

	output := buf.String()
	assert.Contains(t, output, "user action")
	assert.Contains(t, output, "alice")
	assert.Contains(t, output, "login")
	assert.Contains(t, output, "true")
	assert.Contains(t, output, "42")
}

func TestObserver_HandleEvent_EmptyAttributes(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	logger := slog.New(handler)
	observer := NewObserver(logger)
	event := NewEvent(LevelInfo, "simple message", nil)
	observer.HandleEvent(event)

	output := buf.String()
	assert.Contains(t, output, "simple message")
}

func TestObserver_HandleEvent_ConcurrentCalls(t *testing.T) {
	var buf bytes.Buffer
	var mu sync.Mutex

	// Create a simple handler that writes to the buffer
	handler := slog.NewTextHandler(&buf, nil)
	logger := slog.New(handler)
	observer := NewObserver(logger)

	var wg sync.WaitGroup
	numEvents := 50

	wg.Add(numEvents)
	for i := 0; i < numEvents; i++ {
		go func(i int) {
			defer wg.Done()
			event := NewEvent(LevelInfo, "concurrent event", map[string]any{"index": i})
			observer.HandleEvent(event)
		}(i)
	}

	wg.Wait()

	mu.Lock()
	output := buf.String()
	mu.Unlock()

	lines := strings.Split(strings.TrimSpace(output), "\n")
	// Should have at least some output lines
	assert.NotEmpty(t, lines)
}

func TestSlogLevelToLevel(t *testing.T) {
	tests := []struct {
		name      string
		slogLevel slog.Level
		expected  Level
	}{
		{"debug", slog.LevelDebug, LevelDebug},
		{"debug-1", slog.LevelDebug - 1, LevelDebug},
		{"info", slog.LevelInfo, LevelInfo},
		{"warn", slog.LevelWarn, LevelWarn},
		{"error", slog.LevelError, LevelError},
		{"error+1", slog.LevelError + 1, LevelError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := slogLevelToLevel(tt.slogLevel)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLevelToSlogLevel(t *testing.T) {
	tests := []struct {
		name     string
		level    Level
		expected slog.Level
	}{
		{"debug", LevelDebug, slog.LevelDebug},
		{"info", LevelInfo, slog.LevelInfo},
		{"warn", LevelWarn, slog.LevelWarn},
		{"error", LevelError, slog.LevelError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := levelToSlogLevel(tt.level)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestArgsToMap(t *testing.T) {
	tests := []struct {
		name     string
		args     []any
		expected map[string]any
	}{
		{
			name:     "empty args",
			args:     []any{},
			expected: map[string]any{},
		},
		{
			name:     "single key-value pair",
			args:     []any{"key", "value"},
			expected: map[string]any{"key": "value"},
		},
		{
			name:     "multiple pairs",
			args:     []any{"key1", "value1", "key2", 42, "key3", true},
			expected: map[string]any{"key1": "value1", "key2": 42, "key3": true},
		},
		{
			name:     "odd number of args",
			args:     []any{"key1", "value1", "key2"},
			expected: map[string]any{"key1": "value1"},
		},
		{
			name:     "non-string key skipped",
			args:     []any{123, "value", "key2", "value2"},
			expected: map[string]any{"key2": "value2"},
		},
		{
			name:     "nil value",
			args:     []any{"key", nil},
			expected: map[string]any{"key": nil},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := argsToMap(tt.args)
			assert.Equal(t, tt.expected, result)
		})
	}
}
