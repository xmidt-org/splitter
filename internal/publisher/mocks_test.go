// SPDX-FileCopyrightText: 2026 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package publisher

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/xmidt-org/wrp-go/v5"
	"github.com/xmidt-org/wrpkafka"
)

// mockProducer is a producer that records registered listeners so tests can
// emit publish events as the Kafka client would.
type mockProducer struct {
	mock.Mock
	listeners []func(*wrpkafka.PublishEvent)
}

var _ producer = (*mockProducer)(nil)

func (m *mockProducer) Produce(ctx context.Context, msg *wrp.Message) (wrpkafka.Outcome, error) {
	args := m.Called(ctx, msg)
	return args.Get(0).(wrpkafka.Outcome), args.Error(1)
}

func (m *mockProducer) Start() error {
	return m.Called().Error(0)
}

func (m *mockProducer) Stop(ctx context.Context) {
	m.Called(ctx)
}

func (m *mockProducer) AddPublishEventListener(fn func(*wrpkafka.PublishEvent)) func() {
	m.listeners = append(m.listeners, fn)
	return func() { m.listeners = nil }
}

func (m *mockProducer) BufferedRecords() (int, int, int64, int64) {
	args := m.Called()
	return args.Int(0), args.Int(1), int64(args.Int(2)), int64(args.Int(3))
}

// emit delivers a publish event to every registered listener.
func (m *mockProducer) emit(event *wrpkafka.PublishEvent) {
	for _, l := range m.listeners {
		l(event)
	}
}
