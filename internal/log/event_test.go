// SPDX-FileCopyrightText: 2026 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package log

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewEvent_EmptyMessage(t *testing.T) {
	event := NewEvent(LevelWarn, "", nil)

	assert.Equal(t, "", event.Message)
	assert.Equal(t, LevelWarn, event.Level)
}

func TestEvent_AllLevels(t *testing.T) {
	levels := []Level{LevelDebug, LevelInfo, LevelWarn, LevelError}

	for _, level := range levels {
		t.Run(level.String(), func(t *testing.T) {
			event := NewEvent(level, "test", map[string]any{"key": "value"})

			assert.Equal(t, level, event.Level)
			assert.Equal(t, "test", event.Message)
			assert.Equal(t, "value", event.Attrs["key"])
			assert.NotEmpty(t, event.String())
		})
	}
}

func TestEvent_ComplexAttributes(t *testing.T) {
	attrs := map[string]any{
		"string": "value",
		"int":    42,
		"float":  3.14,
		"bool":   true,
		"slice":  []string{"a", "b", "c"},
		"map":    map[string]int{"x": 1, "y": 2},
		"nil":    nil,
	}

	event := NewEvent(LevelInfo, "complex event", attrs)

	assert.Equal(t, "value", event.Attrs["string"])
	assert.Equal(t, 42, event.Attrs["int"])
	assert.Equal(t, 3.14, event.Attrs["float"])
	assert.Equal(t, true, event.Attrs["bool"])
	assert.Equal(t, []string{"a", "b", "c"}, event.Attrs["slice"])
	assert.Equal(t, map[string]int{"x": 1, "y": 2}, event.Attrs["map"])
	assert.Nil(t, event.Attrs["nil"])
}

func TestEvent_Timestamp(t *testing.T) {
	before := time.Now()
	event := NewEvent(LevelInfo, "test", nil)
	after := time.Now()

	assert.True(t, event.Time.After(before) || event.Time.Equal(before))
	assert.True(t, event.Time.Before(after) || event.Time.Equal(after))
}
