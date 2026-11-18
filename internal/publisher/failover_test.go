// SPDX-FileCopyrightText: 2026 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package publisher

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"syscall"
	"testing"
	"time"

	"xmidt-org/splitter/internal/log"
	"xmidt-org/splitter/internal/metrics"
	"xmidt-org/splitter/internal/observe"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/xmidt-org/wrp-go/v5"
	"github.com/xmidt-org/wrpkafka"
)

const (
	testPrimary   = "primary"
	testSecondary = "secondary"
	testOpenWait  = 50 * time.Millisecond
)

var errNotRegionRelated = errors.New("something else")

type FailoverTestSuite struct {
	suite.Suite

	primary   *mockProducer
	secondary *mockProducer
	group     *failoverGroup

	mu           sync.Mutex
	metricEvents []metrics.Event
}

func TestFailoverTestSuite(t *testing.T) {
	suite.Run(t, new(FailoverTestSuite))
}

func (s *FailoverTestSuite) SetupTest() {
	s.metricEvents = nil
	s.primary = &mockProducer{}
	s.secondary = &mockProducer{}

	metricEmitter := observe.NewSubject[metrics.Event]()
	metricEmitter.Attach(func(e metrics.Event) {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.metricEvents = append(s.metricEvents, e)
	})

	cfg := FailoverConfig{
		ConsecutiveFailures: 2,
		OpenTimeout:         testOpenWait,
		HalfOpenSuccesses:   1,
	}
	s.group = newFailoverGroup(log.NewNoop(), metricEmitter)
	s.group.refreshInterval = 5 * time.Millisecond
	s.group.add(testPrimary, s.primary, &cfg)
	s.group.add(testSecondary, s.secondary, &cfg)

	s.primary.On("Start").Return(nil)
	s.secondary.On("Start").Return(nil)
	s.Require().NoError(s.group.start(func(*wrpkafka.PublishEvent) {}))
}

// TearDownTest stops the group's background loop so it cannot outlive the test.
func (s *FailoverTestSuite) TearDownTest() {
	if s.group.stopRefresh != nil {
		s.group.stopRefresh()
		<-s.group.refreshDone
	}
}

// count returns the summed value of events with the given name and labels.
func (s *FailoverTestSuite) count(name string, labels ...string) float64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	var total float64
	for _, e := range s.metricEvents {
		if e.Name == name && labelsEqual(e.Labels, labels) {
			total += e.Value
		}
	}
	return total
}

// last returns the most recent value of the gauge event with the given labels.
func (s *FailoverTestSuite) last(name string, labels ...string) float64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	var value float64
	for _, e := range s.metricEvents {
		if e.Name == name && labelsEqual(e.Labels, labels) {
			value = e.Value
		}
	}
	return value
}

func labelsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func brokerFailure() *wrpkafka.PublishEvent {
	return &wrpkafka.PublishEvent{Error: kgo.ErrRecordTimeout, ErrorType: "unknown"}
}

func (s *FailoverTestSuite) tripPrimary() {
	s.primary.emit(brokerFailure())
	s.primary.emit(brokerFailure())
}

func (s *FailoverTestSuite) TestUsesPrimaryWhenHealthy() {
	s.primary.On("Produce", mock.Anything, mock.Anything).Return(wrpkafka.Accepted, nil)

	outcome, err := s.group.produce(context.Background(), &wrp.Message{})

	s.NoError(err)
	s.Equal(wrpkafka.Accepted, outcome)
	s.secondary.AssertNotCalled(s.T(), "Produce", mock.Anything, mock.Anything)
	s.Zero(s.count(metrics.KafkaRegionFailovers, metrics.FromRegionLabel, testPrimary, metrics.ToRegionLabel, testSecondary))
	s.Equal(1.0, s.last(metrics.KafkaActiveRegion, metrics.RegionLabel, testPrimary))
}

func (s *FailoverTestSuite) TestFailsOverWhenPrimaryBreakerOpens() {
	s.secondary.On("Produce", mock.Anything, mock.Anything).Return(wrpkafka.Accepted, nil)

	s.tripPrimary()
	_, err := s.group.produce(context.Background(), &wrp.Message{})

	s.NoError(err)
	s.secondary.AssertNumberOfCalls(s.T(), "Produce", 1)
	s.primary.AssertNotCalled(s.T(), "Produce", mock.Anything, mock.Anything)

	s.Equal(1.0, s.count(metrics.KafkaRegionFailovers, metrics.FromRegionLabel, testPrimary, metrics.ToRegionLabel, testSecondary))
	s.Equal(1.0, s.count(metrics.KafkaRegionCircuitTransitions,
		metrics.RegionLabel, testPrimary, metrics.FromStateLabel, "closed", metrics.ToStateLabel, "open"))
	s.Equal(2.0, s.last(metrics.KafkaRegionCircuitState, metrics.RegionLabel, testPrimary))
	s.Equal(1.0, s.last(metrics.KafkaActiveRegion, metrics.RegionLabel, testSecondary))
	s.Equal(0.0, s.last(metrics.KafkaActiveRegion, metrics.RegionLabel, testPrimary))
}

func (s *FailoverTestSuite) TestBackgroundLoopTracksStateWithoutTraffic() {
	s.tripPrimary()

	// No messages are produced, yet the failover and the open breaker are reported.
	s.Eventually(func() bool {
		return s.count(metrics.KafkaRegionFailovers, metrics.FromRegionLabel, testPrimary, metrics.ToRegionLabel, testSecondary) == 1 &&
			s.last(metrics.KafkaActiveRegion, metrics.RegionLabel, testSecondary) == 1 &&
			s.last(metrics.KafkaRegionCircuitState, metrics.RegionLabel, testPrimary) == 2
	}, time.Second, time.Millisecond)

	// After the open timeout the breaker is reported half-open and the primary is probed.
	s.Eventually(func() bool {
		return s.last(metrics.KafkaRegionCircuitState, metrics.RegionLabel, testPrimary) == 1 &&
			s.count(metrics.KafkaRegionFailovers, metrics.FromRegionLabel, testSecondary, metrics.ToRegionLabel, testPrimary) == 1
	}, time.Second, time.Millisecond)
}

func (s *FailoverTestSuite) TestFailsBackAfterPrimaryRecovers() {
	s.primary.On("Produce", mock.Anything, mock.Anything).Return(wrpkafka.Accepted, nil)
	s.secondary.On("Produce", mock.Anything, mock.Anything).Return(wrpkafka.Accepted, nil)

	s.tripPrimary()
	_, err := s.group.produce(context.Background(), &wrp.Message{})
	s.Require().NoError(err)

	// After the open timeout the primary is probed again.
	time.Sleep(2 * testOpenWait)
	_, err = s.group.produce(context.Background(), &wrp.Message{})
	s.Require().NoError(err)
	s.primary.AssertNumberOfCalls(s.T(), "Produce", 1)

	// A successful probe closes the breaker.
	s.primary.emit(&wrpkafka.PublishEvent{})
	_, err = s.group.produce(context.Background(), &wrp.Message{})
	s.Require().NoError(err)
	s.primary.AssertNumberOfCalls(s.T(), "Produce", 2)

	s.Equal(1.0, s.count(metrics.KafkaRegionFailovers, metrics.FromRegionLabel, testSecondary, metrics.ToRegionLabel, testPrimary))
	s.Equal(0.0, s.last(metrics.KafkaRegionCircuitState, metrics.RegionLabel, testPrimary))
}

func (s *FailoverTestSuite) TestIgnoresNonRegionErrors() {
	s.primary.On("Produce", mock.Anything, mock.Anything).Return(wrpkafka.Accepted, nil)

	for _, err := range []error{wrpkafka.ErrNoTopicMatch, wrpkafka.ErrEncoding, context.Canceled, kgo.ErrClientClosed, errNotRegionRelated} {
		for i := 0; i < 5; i++ {
			s.primary.emit(&wrpkafka.PublishEvent{Error: err})
		}
	}

	_, err := s.group.produce(context.Background(), &wrp.Message{})
	s.NoError(err)
	s.primary.AssertNumberOfCalls(s.T(), "Produce", 1)
}

func (s *FailoverTestSuite) TestSuccessResetsConsecutiveFailures() {
	s.primary.On("Produce", mock.Anything, mock.Anything).Return(wrpkafka.Accepted, nil)

	s.primary.emit(brokerFailure())
	s.primary.emit(&wrpkafka.PublishEvent{})
	s.primary.emit(brokerFailure())

	_, err := s.group.produce(context.Background(), &wrp.Message{})
	s.NoError(err)
	s.primary.AssertNumberOfCalls(s.T(), "Produce", 1)
}

func (s *FailoverTestSuite) TestAllRegionsOpen() {
	s.tripPrimary()
	s.secondary.emit(brokerFailure())
	s.secondary.emit(brokerFailure())

	outcome, err := s.group.produce(context.Background(), &wrp.Message{})

	s.ErrorIs(err, ErrNoRegionAvailable)
	s.Equal(wrpkafka.Failed, outcome)
	s.Eventually(func() bool {
		return s.count(metrics.PublisherErrorsCounter,
			metrics.ErrorTypeLabel, "no_region_available",
			metrics.TopicLabel, "unknown",
			metrics.TopicShardStrategyLabel, "unknown") == 1
	}, time.Second, time.Millisecond)
}

func (s *FailoverTestSuite) TestSingleRegionWithoutBreaker() {
	p := &mockProducer{}
	p.On("Produce", mock.Anything, mock.Anything).Return(wrpkafka.Failed, errNotRegionRelated)
	g := newFailoverGroup(log.NewNoop(), metrics.NewNoop())
	g.add(testPrimary, p, nil)

	for i := 0; i < 10; i++ {
		_, err := g.produce(context.Background(), &wrp.Message{})
		s.ErrorIs(err, errNotRegionRelated)
	}
	p.AssertNumberOfCalls(s.T(), "Produce", 10)
}

func (s *FailoverTestSuite) TestStartFailureStopsStartedRegions() {
	primary, secondary := &mockProducer{}, &mockProducer{}
	primary.On("Start").Return(nil)
	primary.On("Stop", mock.Anything).Return()
	secondary.On("Start").Return(errNotRegionRelated)
	g := newFailoverGroup(log.NewNoop(), metrics.NewNoop())
	g.add(testPrimary, primary, nil)
	g.add(testSecondary, secondary, nil)

	err := g.start(func(*wrpkafka.PublishEvent) {})

	s.ErrorIs(err, errNotRegionRelated)
	primary.AssertCalled(s.T(), "Stop", mock.Anything)
	s.Empty(primary.listeners)
}

func (s *FailoverTestSuite) TestBufferedRecordsSumsRegions() {
	s.primary.On("BufferedRecords").Return(1, 10, 100, 1000)
	s.secondary.On("BufferedRecords").Return(2, 20, 200, 2000)

	cr, mr, cb, mb := s.group.bufferedRecords()

	s.Equal([]int{3, 30}, []int{cr, mr})
	s.Equal([]int64{300, 3000}, []int64{cb, mb})
}

func (s *FailoverTestSuite) TestStopStopsAllRegions() {
	s.primary.On("Stop", mock.Anything).Return()
	s.secondary.On("Stop", mock.Anything).Return()

	s.group.stop(context.Background())

	s.primary.AssertNumberOfCalls(s.T(), "Stop", 1)
	s.secondary.AssertNumberOfCalls(s.T(), "Stop", 1)
}

func TestIsRegionFailure(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"record timeout", kgo.ErrRecordTimeout, true},
		{"retries exhausted", kgo.ErrRecordRetries, true},
		{"buffer full", kgo.ErrMaxBuffered, true},
		{"wrapped broker error", fmt.Errorf("broker rejected message: %w", kerr.RequestTimedOut), true},
		{"leader unavailable", kerr.LeaderNotAvailable, true},
		{"connection refused", &net.OpError{Op: "dial", Err: syscall.ECONNREFUSED}, true},
		{"unexpected EOF", io.ErrUnexpectedEOF, true},
		{"deadline exceeded", context.DeadlineExceeded, true},
		{"message too large", kerr.MessageTooLarge, false},
		{"topic authorization", kerr.TopicAuthorizationFailed, false},
		{"client closed", kgo.ErrClientClosed, false},
		{"aborting", kgo.ErrAborting, false},
		{"caller canceled", context.Canceled, false},
		{"no topic match", wrpkafka.ErrNoTopicMatch, false},
		{"unrecognized", errNotRegionRelated, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRegionFailure(tt.err); got != tt.want {
				t.Fatalf("isRegionFailure(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestBrokersFailoverRegionOrder(t *testing.T) {
	regions := map[string][]string{"a": {"a:1"}, "b": {"b:1"}, "c": {"c:1"}}

	tests := []struct {
		name    string
		brokers Brokers
		want    []string
		wantErr string
	}{
		{
			name:    "disabled uses target only",
			brokers: Brokers{TargetRegion: "b", Regions: regions},
			want:    []string{"b"},
		},
		{
			name:    "enabled defaults to other regions alphabetically",
			brokers: Brokers{TargetRegion: "b", Regions: regions, Failover: FailoverConfig{Enabled: true}},
			want:    []string{"b", "a", "c"},
		},
		{
			name:    "explicit order",
			brokers: Brokers{TargetRegion: "a", Regions: regions, Failover: FailoverConfig{Enabled: true, Regions: []string{"c", "b"}}},
			want:    []string{"a", "c", "b"},
		},
		{
			name:    "no failover region available",
			brokers: Brokers{TargetRegion: "a", Regions: map[string][]string{"a": {"a:1"}}, Failover: FailoverConfig{Enabled: true}},
			wantErr: ErrNoFailoverRegions.Error(),
		},
		{
			name:    "unknown failover region",
			brokers: Brokers{TargetRegion: "a", Regions: regions, Failover: FailoverConfig{Enabled: true, Regions: []string{"x"}}},
			wantErr: "failover region 'x' not found in regions map",
		},
		{
			name:    "failover region equals target",
			brokers: Brokers{TargetRegion: "a", Regions: regions, Failover: FailoverConfig{Enabled: true, Regions: []string{"a"}}},
			wantErr: "failover region 'a' is listed more than once or equals target_region",
		},
		{
			name: "failover region without brokers",
			brokers: Brokers{TargetRegion: "a", Regions: map[string][]string{"a": {"a:1"}, "b": {}},
				Failover: FailoverConfig{Enabled: true}},
			wantErr: "failover region 'b' has no brokers configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.brokers.validateFailover()
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("validateFailover() error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("validateFailover() unexpected error: %v", err)
			}
			got := tt.brokers.regionOrder()
			if len(got) != len(tt.want) {
				t.Fatalf("regionOrder() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("regionOrder() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}
