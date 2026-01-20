// SPDX-FileCopyrightText: 2026 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package log

import (
	"log/slog"
	"xmidt-org/splitter/internal/observe"
)

// New creates a new Subject for logging events
func New(logger *slog.Logger) *observe.Subject[Event] {
	subject := observe.NewSubject[Event]()
	observer := NewObserver(logger)
	subject.Attach(observer.HandleEvent)

	return subject
}

func NewNoop() *observe.Subject[Event] {
	s := observe.NewSubject[Event]()
	// Don't attach any observers - events will be discarded
	return s
}
