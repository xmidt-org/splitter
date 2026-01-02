// SPDX-FileCopyrightText: 2026 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package log

import (
	"time"
)

// Level represents the severity level of a log event.
type Level int

const (
	// LevelDebug is the debug level
	LevelDebug Level = -4
	// LevelInfo is the info level
	LevelInfo Level = 0
	// LevelWarn is the warn level
	LevelWarn Level = 4
	// LevelError is the error level
	LevelError Level = 8
)

// String returns the string representation of the level.
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Event represents a generic structured log event that is not tied to a specific logger.
// It captures all the information about a log entry for observation and processing.
type Event struct {
	// Time is when the log event occurred
	Time time.Time

	// Level is the severity level of the log event
	Level Level

	// Message is the log message
	Message string

	// Attributes contains the structured key-value pairs associated with the log
	Attrs map[string]any
}

// NewEvent creates a new Event with the current timestamp.
func NewEvent(level Level, message string, attrs map[string]any) Event {
	if attrs == nil {
		attrs = make(map[string]any)
	}
	return Event{
		Time:    time.Now(),
		Level:   level,
		Message: message,
		Attrs:   attrs,
	}
}

// String returns a simple string representation of the log event.
func (e Event) String() string {
	return e.Level.String() + ": " + e.Message
}
