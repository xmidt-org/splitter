// SPDX-FileCopyrightText: 2026 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"testing"

	kit "github.com/go-kit/kit/metrics"
	"github.com/stretchr/testify/suite"
)

type ObserverTestSuite struct {
	suite.Suite
}

func TestObserverTestSuite(t *testing.T) {
	suite.Run(t, new(ObserverTestSuite))
}

// mockCounter is a mock implementation of kit.Counter
type mockCounter struct {
	addCalled  bool
	addValue   float64
	withLabels []string
}

func (m *mockCounter) With(labelValues ...string) kit.Counter {
	// Store the labels and return self to track the Add call
	m.withLabels = append([]string{}, labelValues...)
	return m
}

func (m *mockCounter) Add(delta float64) {
	m.addCalled = true
	m.addValue += delta
}

// mockGauge is a mock implementation of kit.Gauge
type mockGauge struct {
	setCalled  bool
	setValue   float64
	addCalled  bool
	addValue   float64
	withLabels []string
}

func (m *mockGauge) With(labelValues ...string) kit.Gauge {
	m.withLabels = append([]string{}, labelValues...)
	return m
}

func (m *mockGauge) Set(value float64) {
	m.setCalled = true
	m.setValue = value
}

func (m *mockGauge) Add(delta float64) {
	m.addCalled = true
	m.addValue += delta
}

// mockHistogram is a mock implementation of kit.Histogram
type mockHistogram struct {
	observeCalled bool
	observeValue  float64
	withLabels    []string
}

func (m *mockHistogram) With(labelValues ...string) kit.Histogram {
	m.withLabels = append([]string{}, labelValues...)
	return m
}

func (m *mockHistogram) Observe(value float64) {
	m.observeCalled = true
	m.observeValue = value
}

// TestNewObserver tests creating new observers
func (s *ObserverTestSuite) TestNewObserver() {
	tests := []struct {
		name         string
		metricName   string
		metricType   metricType
		expectedName string
		expectedType metricType
		hasCounter   bool
		hasGauge     bool
		hasHistogram bool
	}{
		{
			name:         "counter observer",
			metricName:   "test_counter",
			metricType:   COUNTER,
			expectedName: "test_counter",
			expectedType: COUNTER,
			hasCounter:   true,
		},
		{
			name:         "gauge observer",
			metricName:   "test_gauge",
			metricType:   GAUGE,
			expectedName: "test_gauge",
			expectedType: GAUGE,
			hasGauge:     true,
		},
		{
			name:         "histogram observer",
			metricName:   "test_histogram",
			metricType:   HISTOGRAM,
			expectedName: "test_histogram",
			expectedType: HISTOGRAM,
			hasHistogram: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var metric Metric
			if tt.hasCounter {
				metric.counter = &mockCounter{}
			}
			if tt.hasGauge {
				metric.gauge = &mockGauge{}
			}
			if tt.hasHistogram {
				metric.histogram = &mockHistogram{}
			}

			observer := NewObserver(tt.metricName, tt.metricType, metric)

			s.NotNil(observer)
			s.Equal(tt.expectedName, observer.name)
			s.Equal(tt.expectedType, observer.metricType)
		})
	}
}

// TestObserver_HandleEvent_Counter tests handling counter events
func (s *ObserverTestSuite) TestObserver_HandleEvent_Counter() {
	counter := &mockCounter{}
	observer := NewObserver("test_counter", COUNTER, Metric{counter: counter})

	event := Event{
		Name:   "test_counter",
		Labels: []string{"label1", "value1"},
		Value:  5.0,
	}

	observer.HandleEvent(event)

	s.True(counter.addCalled)
	s.Equal(5.0, counter.addValue)
	s.Equal([]string{"label1", "value1"}, counter.withLabels)
}

// TestObserver_HandleEvent_Gauge tests handling gauge events
func (s *ObserverTestSuite) TestObserver_HandleEvent_Gauge() {
	gauge := &mockGauge{}
	observer := NewObserver("test_gauge", GAUGE, Metric{gauge: gauge})

	event := Event{
		Name:   "test_gauge",
		Labels: []string{"label1", "value1"},
		Value:  42.5,
	}

	observer.HandleEvent(event)

	s.True(gauge.setCalled)
	s.Equal(42.5, gauge.setValue)
	s.Equal([]string{"label1", "value1"}, gauge.withLabels)
}

// TestObserver_HandleEvent_Histogram tests handling histogram events
func (s *ObserverTestSuite) TestObserver_HandleEvent_Histogram() {
	histogram := &mockHistogram{}
	observer := NewObserver("test_histogram", HISTOGRAM, Metric{histogram: histogram})

	event := Event{
		Name:   "test_histogram",
		Labels: []string{"label1", "value1"},
		Value:  0.125,
	}

	observer.HandleEvent(event)

	s.True(histogram.observeCalled)
	s.Equal(0.125, histogram.observeValue)
	s.Equal([]string{"label1", "value1"}, histogram.withLabels)
}

// TestObserver_HandleEvent_WrongName tests that events with wrong names are ignored
func (s *ObserverTestSuite) TestObserver_HandleEvent_WrongName() {
	counter := &mockCounter{}
	observer := NewObserver("test_counter", COUNTER, Metric{counter: counter})

	event := Event{
		Name:   "different_counter",
		Labels: []string{"label1", "value1"},
		Value:  5.0,
	}

	observer.HandleEvent(event)

	// Should not be called because name doesn't match
	s.False(counter.addCalled)
}

// TestObserver_HandleEvent_NilCounter tests handling events when counter is nil
func (s *ObserverTestSuite) TestObserver_HandleEvent_NilCounter() {
	observer := NewObserver("test_counter", COUNTER, Metric{})

	event := Event{
		Name:   "test_counter",
		Labels: []string{},
		Value:  1.0,
	}

	// Should not panic, just print error
	s.NotPanics(func() {
		observer.HandleEvent(event)
	})
}

// TestObserver_HandleEvent_NilGauge tests handling events when gauge is nil
func (s *ObserverTestSuite) TestObserver_HandleEvent_NilGauge() {
	observer := NewObserver("test_gauge", GAUGE, Metric{})

	event := Event{
		Name:   "test_gauge",
		Labels: []string{},
		Value:  1.0,
	}

	// Should not panic, just print error
	s.NotPanics(func() {
		observer.HandleEvent(event)
	})
}

// TestObserver_HandleEvent_NilHistogram tests handling events when histogram is nil
func (s *ObserverTestSuite) TestObserver_HandleEvent_NilHistogram() {
	observer := NewObserver("test_histogram", HISTOGRAM, Metric{})

	event := Event{
		Name:   "test_histogram",
		Labels: []string{},
		Value:  1.0,
	}

	// Should not panic, just print error
	s.NotPanics(func() {
		observer.HandleEvent(event)
	})
}

// TestObserver_HandleEvent_EmptyLabels tests handling events with empty labels
func (s *ObserverTestSuite) TestObserver_HandleEvent_EmptyLabels() {
	counter := &mockCounter{}
	observer := NewObserver("test_counter", COUNTER, Metric{counter: counter})

	event := Event{
		Name:   "test_counter",
		Labels: []string{},
		Value:  10.0,
	}

	observer.HandleEvent(event)

	s.True(counter.addCalled)
	s.Equal(10.0, counter.addValue)
	s.Equal([]string{}, counter.withLabels)
}

// TestObserver_HandleEvent_MultipleLabels tests handling events with multiple label pairs
func (s *ObserverTestSuite) TestObserver_HandleEvent_MultipleLabels() {
	counter := &mockCounter{}
	observer := NewObserver(ConsumerErrors, COUNTER, Metric{counter: counter})

	event := Event{
		Name:   ConsumerErrors,
		Labels: []string{PartitionLabel, "0", TopicLabel, "test-topic", ErrorTypeLabel, "decode_error"},
		Value:  1.0,
	}

	observer.HandleEvent(event)

	s.True(counter.addCalled)
	s.Equal(1.0, counter.addValue)
	s.Contains(counter.withLabels, PartitionLabel)
	s.Contains(counter.withLabels, TopicLabel)
	s.Contains(counter.withLabels, ErrorTypeLabel)
}
