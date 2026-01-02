// SPDX-FileCopyrightText: 2026 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package log

import (
	"context"
	"log/slog"
)

type Observer struct {
	logger *slog.Logger
}

// New creates a new Observer that wraps the given slog.Logger.
// The subject will emit events to the provided emitter.
func NewObserver(logger *slog.Logger) *Observer {
	return &Observer{
		logger: logger,
	}
}

func (o *Observer) HandleEvent(event Event) {
	attrs := make([]slog.Attr, 0, len(event.Attrs))
	for k, v := range event.Attrs {
		attrs = append(attrs, slog.Any(k, v))
	}
	o.logger.LogAttrs(context.Background(), levelToSlogLevel(event.Level), event.Message, attrs...)
}

func slogLevelToLevel(sl slog.Level) Level {
	switch {
	case sl < slog.LevelInfo:
		return LevelDebug
	case sl < slog.LevelWarn:
		return LevelInfo
	case sl < slog.LevelError:
		return LevelWarn
	default:
		return LevelError
	}
}

func levelToSlogLevel(sl Level) slog.Level {
	switch {
	case sl < LevelInfo:
		return slog.LevelDebug
	case sl < LevelWarn:
		return slog.LevelInfo
	case sl < LevelError:
		return slog.LevelWarn
	default:
		return slog.LevelError
	}
}

// argsToMap converts variadic args to a map.
func argsToMap(args []any) map[string]any {
	attrs := make(map[string]any)
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			key, ok := args[i].(string)
			if ok {
				attrs[key] = args[i+1]
			}
		}
	}
	return attrs
}
