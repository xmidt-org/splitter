// SPDX-FileCopyrightText: 2026 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package consumer

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/twmb/franz-go/pkg/kgo"

	"xmidt-org/splitter/internal/log"
	"xmidt-org/splitter/internal/metrics"
)

// ConsumerTestSuite is the test suite for the Consumer
type ConsumerTestSuite struct {
	suite.Suite
	mock    *mockClient
	handler MessageHandler
}

// SetupTest is called before each test
func (s *ConsumerTestSuite) SetupTest() {
	s.mock = newMockClient()
	s.handler = MessageHandlerFunc(func(ctx context.Context, record *kgo.Record) error {
		return nil
	})
}

// TestNew_RequiredOptions tests that New validates required options
func (s *ConsumerTestSuite) TestNew_RequiredOptions() {
	tests := []struct {
		name        string
		opts        []Option
		expectedErr string
	}{
		{
			name:        "missing all required options",
			opts:        []Option{},
			expectedErr: "brokers are required",
		},
		{
			name: "missing topics",
			opts: []Option{
				WithBrokers("localhost:9092"),
			},
			expectedErr: "topics are required",
		},
		{
			name: "missing group ID",
			opts: []Option{
				WithBrokers("localhost:9092"),
				WithTopics("test-topic"),
			},
			expectedErr: "group ID is required",
		},
		{
			name: "missing handler",
			opts: []Option{
				WithBrokers("localhost:9092"),
				WithTopics("test-topic"),
				WithGroupID("test-group"),
			},
			expectedErr: "message handler is required",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			c, err := New(tt.opts...)
			s.Error(err)
			s.Nil(c)
			s.Contains(err.Error(), tt.expectedErr)
		})
	}
}

// TestNew_ValidConfiguration tests successful consumer creation
func (s *ConsumerTestSuite) TestNew_ValidConfiguration() {
	c, err := New(
		WithBrokers("localhost:9092"),
		WithTopics("test-topic"),
		WithGroupID("test-group"),
		WithMessageHandler(s.handler),
	)

	s.NoError(err)
	s.NotNil(c)
	s.False(c.IsRunning())
}

// TestNew_WithOptionalOptions tests consumer creation with optional options
func (s *ConsumerTestSuite) TestNew_WithOptionalOptions() {
	c, err := New(
		WithBrokers("localhost:9092", "localhost:9093"),
		WithTopics("topic1", "topic2"),
		WithGroupID("test-group"),
		WithMessageHandler(s.handler),
		WithSessionTimeout(60*time.Second),
		WithClientID("test-client"),
	)

	s.NoError(err)
	s.NotNil(c)
}

// TestConsumer_IsRunning tests the IsRunning method
func (s *ConsumerTestSuite) TestConsumer_IsRunning() {
	c := &KafkaConsumer{}
	s.False(c.IsRunning())

	c.mu.Lock()
	c.running = true
	c.mu.Unlock()

	s.True(c.IsRunning())
}

// TestConsumer_Start_Success tests successful consumer start
func (s *ConsumerTestSuite) TestConsumer_Start_Success() {
	c := &KafkaConsumer{
		client:  s.mock,
		handler: s.handler,
		config:  &consumerConfig{},
	}
	c.ctx, c.cancel = context.WithCancel(context.Background())
	c.logEmitter = log.NewNoop()
	c.metricEmitter = metrics.NewNoop()

	err := c.Start()
	s.NoError(err)

	// Give pollLoop time to start
	time.Sleep(50 * time.Millisecond)

	s.True(c.IsRunning())
	s.True(s.mock.wasPingCalled())

	// Cleanup
	err = c.Stop(context.Background())
	s.NoError(err)
}

// TestConsumer_Start_AlreadyRunning tests starting an already running consumer
func (s *ConsumerTestSuite) TestConsumer_Start_AlreadyRunning() {
	c := &KafkaConsumer{
		client:  s.mock,
		handler: s.handler,
		config:  &consumerConfig{},
	}
	c.ctx, c.cancel = context.WithCancel(context.Background())
	c.logEmitter = log.NewNoop()
	c.metricEmitter = metrics.NewNoop()

	err := c.Start()
	s.NoError(err)
	defer c.Stop(context.Background())

	// Try to start again
	err = c.Start()
	s.Error(err)
	s.Contains(err.Error(), "consumer is already running")
}

// TestConsumer_Start_PingFailure tests start with ping failure
func (s *ConsumerTestSuite) TestConsumer_Start_PingFailure() {
	s.mock.pingFn = func(ctx context.Context) error {
		return errors.New("ping failed")
	}

	c := &KafkaConsumer{
		client:  s.mock,
		handler: s.handler,
		config:  &consumerConfig{},
	}
	c.ctx, c.cancel = context.WithCancel(context.Background())
	c.logEmitter = log.NewNoop()
	c.metricEmitter = metrics.NewNoop()

	err := c.Start()
	s.Error(err)
	s.False(c.IsRunning())
}

// TestConsumer_Stop tests graceful consumer shutdown
func (s *ConsumerTestSuite) TestConsumer_Stop() {
	c := &KafkaConsumer{
		client:  s.mock,
		handler: s.handler,
		config:  &consumerConfig{},
	}
	c.ctx, c.cancel = context.WithCancel(context.Background())
	c.logEmitter = log.NewNoop()
	c.metricEmitter = metrics.NewNoop()

	err := c.Start()
	s.NoError(err)

	// Give pollLoop time to start
	time.Sleep(50 * time.Millisecond)

	err = c.Stop(context.Background())
	s.NoError(err)

	s.False(c.IsRunning())
	s.True(s.mock.wasCloseCalled())
}

// TestConsumer_Stop_NotRunning tests stopping a non-running consumer
func (s *ConsumerTestSuite) TestConsumer_Stop_NotRunning() {
	c := &KafkaConsumer{
		client:  s.mock,
		handler: s.handler,
		config:  &consumerConfig{},
	}
	c.ctx, c.cancel = context.WithCancel(context.Background())
	c.logEmitter = log.NewNoop()
	c.metricEmitter = metrics.NewNoop()

	// Stop without starting should return error
	err := c.Stop(context.Background())
	s.Error(err)
	s.Contains(err.Error(), "consumer is not running")
	s.False(c.IsRunning())
}

// TestConsumer_HandleRecord_Success tests successful message handling
func (s *ConsumerTestSuite) TestConsumer_HandleRecord_Success() {
	handlerCalled := false
	var receivedRecord *kgo.Record

	handler := MessageHandlerFunc(func(ctx context.Context, record *kgo.Record) error {
		handlerCalled = true
		receivedRecord = record
		return nil
	})

	c := &KafkaConsumer{
		client:  s.mock,
		handler: handler,
		config:  &consumerConfig{},
	}
	c.ctx, c.cancel = context.WithCancel(context.Background())
	c.logEmitter = log.NewNoop()
	c.metricEmitter = metrics.NewNoop()

	record := &kgo.Record{
		Topic:     "test-topic",
		Partition: 0,
		Offset:    123,
		Value:     []byte("test-value"),
	}

	c.handleRecord(record)

	s.True(handlerCalled)
	s.Equal(record, receivedRecord)
}

// TestConsumer_HandleRecord_Error tests message handling with errors
func (s *ConsumerTestSuite) TestConsumer_HandleRecord_Error() {
	testErr := errors.New("handler error")

	handler := MessageHandlerFunc(func(ctx context.Context, record *kgo.Record) error {
		return testErr
	})

	c := &KafkaConsumer{
		client:  s.mock,
		handler: handler,
		config:  &consumerConfig{},
	}
	c.ctx, c.cancel = context.WithCancel(context.Background())
	c.logEmitter = log.NewNoop()
	c.metricEmitter = metrics.NewNoop()

	record := &kgo.Record{
		Topic: "test-topic",
	}

	// Should not panic even with handler error
	s.NotPanics(func() {
		c.handleRecord(record)
	})
}

// TestConsumer_PollLoop_WithRecords tests the poll loop with records
func (s *ConsumerTestSuite) TestConsumer_PollLoop_WithRecords() {
	handlerCallCount := 0
	var mu sync.Mutex

	handler := MessageHandlerFunc(func(ctx context.Context, record *kgo.Record) error {
		mu.Lock()
		handlerCallCount++
		mu.Unlock()
		return nil
	})

	mock := newMockClient()

	// Create a fetch with records
	pollCount := 0
	mock.pollFetchesFn = func(ctx context.Context) kgo.Fetches {
		pollCount++
		if pollCount == 1 {
			// Return records on first poll
			fetches := kgo.Fetches{}
			fetches = append(fetches, kgo.Fetch{
				Topics: []kgo.FetchTopic{
					{
						Topic: "test-topic",
						Partitions: []kgo.FetchPartition{
							{
								Partition: 0,
								Records: []*kgo.Record{
									{Topic: "test-topic", Partition: 0, Offset: 1, Value: []byte("msg1")},
									{Topic: "test-topic", Partition: 0, Offset: 2, Value: []byte("msg2")},
								},
							},
						},
					},
				},
			})
			return fetches
		}
		// Return empty on subsequent polls to allow test to finish
		return kgo.Fetches{}
	}

	c := &KafkaConsumer{
		client:  mock,
		handler: handler,
		config:  &consumerConfig{},
	}
	c.ctx, c.cancel = context.WithCancel(context.Background())
	c.logEmitter = log.NewNoop()
	c.metricEmitter = metrics.NewNoop()

	err := c.Start()
	s.NoError(err)

	// Give pollLoop time to process
	time.Sleep(100 * time.Millisecond)

	err = c.Stop(context.Background())
	s.NoError(err)

	// Should have processed 2 records
	mu.Lock()
	count := handlerCallCount
	mu.Unlock()
	s.Equal(2, count)
	s.True(mock.getPollFetchesCallCount() > 0)
}

// TestConsumerTestSuite runs the test suite
func TestConsumerTestSuite(t *testing.T) {
	suite.Run(t, new(ConsumerTestSuite))
}
