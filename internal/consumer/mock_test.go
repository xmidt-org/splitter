// SPDX-FileCopyrightText: 2026 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package consumer

import (
	"context"
	"sync"

	"github.com/twmb/franz-go/pkg/kgo"
)

// mockClient is a mock implementation of the Client interface for testing.
type mockClient struct {
	mu sync.Mutex

	// Function hooks for test control
	pingFn                     func(ctx context.Context) error
	pollFetchesFn              func(ctx context.Context) kgo.Fetches
	commitUncommittedOffsetsFn func(ctx context.Context) error
	closeFn                    func()

	// State tracking
	pingCalled                     bool
	pollFetchesCallCount           int
	commitUncommittedOffsetsCalled bool
	closeCalled                    bool
}

// newMockClient creates a new mock client with default no-op implementations.
func newMockClient() *mockClient {
	return &mockClient{
		pingFn: func(ctx context.Context) error {
			return nil
		},
		pollFetchesFn: func(ctx context.Context) kgo.Fetches {
			return kgo.Fetches{}
		},
		commitUncommittedOffsetsFn: func(ctx context.Context) error {
			return nil
		},
		closeFn: func() {},
	}
}

func (m *mockClient) Ping(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pingCalled = true
	return m.pingFn(ctx)
}

func (m *mockClient) PollFetches(ctx context.Context) kgo.Fetches {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pollFetchesCallCount++
	return m.pollFetchesFn(ctx)
}

func (m *mockClient) CommitUncommittedOffsets(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.commitUncommittedOffsetsCalled = true
	return m.commitUncommittedOffsetsFn(ctx)
}

func (m *mockClient) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closeCalled = true
	m.closeFn()
}

// Helper methods for test assertions
func (m *mockClient) wasPingCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.pingCalled
}

func (m *mockClient) getPollFetchesCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.pollFetchesCallCount
}

func (m *mockClient) wasCommitCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.commitUncommittedOffsetsCalled
}

func (m *mockClient) wasCloseCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closeCalled
}
