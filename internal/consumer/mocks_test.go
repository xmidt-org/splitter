// SPDX-FileCopyrightText: 2026 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package consumer

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/xmidt-org/wrp-go/v5"
	"github.com/xmidt-org/wrpkafka"
)

// MockClient is a mock implementation of the Client interface.
//
// Note: This mock does NOT use an additional mutex beyond what testify's mock.Mock provides.
// Adding a mutex caused data races on GitHub Actions because:
// 1. The mock would lock and inspect method arguments (including context) for logging
// 2. While locked, if the test goroutine called context.Cancel(), it would modify the context
// 3. This created a race between reading (mock formatting) and writing (cancel modifying context state)
//
// Solution: testify's mock.Mock already has internal synchronization, and we use mock.MatchedBy
// with simple functions for context parameters to avoid deep inspection during concurrent cancellation.
type MockClient struct {
	mock.Mock
}

func (m *MockClient) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockClient) PollFetches(ctx context.Context) Fetches {
	args := m.Called(ctx)
	return args.Get(0).(Fetches)
}

func (m *MockClient) MarkCommitRecords(records ...*kgo.Record) {
	m.Called(records)
}

func (m *MockClient) CommitUncommittedOffsets(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockClient) PauseFetchTopics(topics ...string) []string {
	args := m.Called(topics)
	if len(args) == 0 {
		return nil
	}
	if result := args.Get(0); result != nil {
		return result.([]string)
	}
	return nil
}

func (m *MockClient) ResumeFetchTopics(topics ...string) {
	m.Called(topics)
}

func (m *MockClient) Close() {
	m.Called()
}

func (m *MockClient) CommitMarkedOffsets(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// MockHandler is a mock implementation of the MessageHandler interface
type MockHandler struct {
	mock.Mock
}

func (m *MockHandler) HandleMessage(ctx context.Context, record *kgo.Record) (Outcome, error) {
	args := m.Called(ctx, record)
	return args.Get(0).(Outcome), args.Error(1)
}

// MockFetches is a testify mock for the Fetches interface.
type MockFetches struct {
	mock.Mock
}

func (m *MockFetches) Errors() []*kgo.FetchError {
	args := m.Called()
	return args.Get(0).([]*kgo.FetchError)
}

func (m *MockFetches) EachRecord(fn func(*kgo.Record)) {
	m.Called(fn)
}

type MockBuckets struct {
	mock.Mock
}

func (m *MockBuckets) IsInTargetBucket(msg *wrp.Message) bool {
	args := m.Called(msg)
	return args.Bool(0)
}

// MockPublisher implements publisher.Publisher for testing
type MockPublisher struct {
	mock.Mock
}

func (m *MockPublisher) Produce(ctx context.Context, msg *wrp.Message) (wrpkafka.Outcome, error) {
	args := m.Called(ctx, msg)
	return args.Get(0).(wrpkafka.Outcome), args.Error(1)
}
func (m *MockPublisher) Start() error                   { return nil }
func (m *MockPublisher) Stop(ctx context.Context) error { return nil }
