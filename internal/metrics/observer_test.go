// SPDX-FileCopyrightText: 2026 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"testing"

	kit "github.com/go-kit/kit/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCounterObserver(t *testing.T) {
	tests := []struct {
		name          string
		setupCounters func() map[string]kit.Counter
		events        []Event
		shouldPanic   bool
		expectPanic   bool
		verifyMocks   bool
	}{
		{
			name: "handle multiple counter events",
			setupCounters: func() map[string]kit.Counter {
				counter1 := &MockCounter{}
				counter1.On("With", []string{"label1", "value1"}).Return(counter1)
				counter1.On("Add", 5.0).Return()

				counter2 := &MockCounter{}
				counter2.On("With", []string{"label2", "value2"}).Return(counter2)
				counter2.On("Add", 10.0).Return()

				return map[string]kit.Counter{
					"test_counter_1": counter1,
					"test_counter_2": counter2,
				}
			},
			events: []Event{
				{Name: "test_counter_1", Labels: []string{"label1", "value1"}, Value: 5.0},
				{Name: "test_counter_2", Labels: []string{"label2", "value2"}, Value: 10.0},
				{Name: "unknown_counter", Labels: []string{"label3", "value3"}, Value: 15.0},
			},
			verifyMocks: true,
		},
		{
			name: "handle nil counter gracefully",
			setupCounters: func() map[string]kit.Counter {
				return map[string]kit.Counter{
					"test_counter": nil,
				}
			},
			events: []Event{
				{Name: "test_counter", Labels: []string{}, Value: 1.0},
			},
			shouldPanic: false,
		},
		{
			name: "handle empty counters map",
			setupCounters: func() map[string]kit.Counter {
				return map[string]kit.Counter{}
			},
			events: []Event{
				{Name: "any_counter", Labels: []string{}, Value: 1.0},
			},
			shouldPanic: false,
		},
		{
			name: "recover from panic gracefully",
			setupCounters: func() map[string]kit.Counter {
				counter := &MockCounter{}
				counter.On("With", []string{"label1", "value1"}).Return(counter)
				counter.On("Add", 5.0).Run(func(args mock.Arguments) {
					panic("test panic")
				})
				return map[string]kit.Counter{
					"test_counter": counter,
				}
			},
			events: []Event{
				{Name: "test_counter", Labels: []string{"label1", "value1"}, Value: 5.0},
			},
			expectPanic: false,
			verifyMocks: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counters := tt.setupCounters()
			observer := NewCounterObserver(counters)

			for _, event := range tt.events {
				if tt.expectPanic {
					assert.NotPanics(t, func() {
						observer.HandleEvent(event)
					})
				} else if tt.shouldPanic {
					assert.NotPanics(t, func() {
						observer.HandleEvent(event)
					})
				} else {
					assert.NotPanics(t, func() {
						observer.HandleEvent(event)
					})
				}
			}

			if tt.verifyMocks {
				for _, counter := range counters {
					if mockCounter, ok := counter.(*MockCounter); ok {
						mockCounter.AssertExpectations(t)
					}
				}
			}
		})
	}
}

func TestGaugeObserver(t *testing.T) {
	tests := []struct {
		name        string
		setupGauges func() map[string]kit.Gauge
		events      []Event
		shouldPanic bool
		verifyMocks bool
	}{
		{
			name: "handle multiple gauge events",
			setupGauges: func() map[string]kit.Gauge {
				gauge1 := &MockGauge{}
				gauge1.On("With", []string{"label1", "value1"}).Return(gauge1)
				gauge1.On("Set", 42.5).Return()

				gauge2 := &MockGauge{}
				gauge2.On("With", []string{"label2", "value2"}).Return(gauge2)
				gauge2.On("Set", 100.0).Return()

				return map[string]kit.Gauge{
					"test_gauge_1": gauge1,
					"test_gauge_2": gauge2,
				}
			},
			events: []Event{
				{Name: "test_gauge_1", Labels: []string{"label1", "value1"}, Value: 42.5},
				{Name: "test_gauge_2", Labels: []string{"label2", "value2"}, Value: 100.0},
				{Name: "unknown_gauge", Labels: []string{"label3", "value3"}, Value: 200.0},
			},
			verifyMocks: true,
		},
		{
			name: "handle nil gauge gracefully",
			setupGauges: func() map[string]kit.Gauge {
				return map[string]kit.Gauge{
					"test_gauge": nil,
				}
			},
			events: []Event{
				{Name: "test_gauge", Labels: []string{}, Value: 1.0},
			},
			shouldPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gauges := tt.setupGauges()
			observer := NewGaugeObserver(gauges)

			for _, event := range tt.events {
				if tt.shouldPanic {
					assert.NotPanics(t, func() {
						observer.HandleEvent(event)
					})
				} else {
					assert.NotPanics(t, func() {
						observer.HandleEvent(event)
					})
				}
			}

			if tt.verifyMocks {
				for _, gauge := range gauges {
					if mockGauge, ok := gauge.(*MockGauge); ok {
						mockGauge.AssertExpectations(t)
					}
				}
			}
		})
	}
}

func TestHistogramObserver(t *testing.T) {
	tests := []struct {
		name            string
		setupHistograms func() map[string]kit.Histogram
		events          []Event
		shouldPanic     bool
		verifyMocks     bool
	}{
		{
			name: "handle multiple histogram events",
			setupHistograms: func() map[string]kit.Histogram {
				histogram1 := &MockHistogram{}
				histogram1.On("With", []string{"label1", "value1"}).Return(histogram1)
				histogram1.On("Observe", 0.125).Return()

				histogram2 := &MockHistogram{}
				histogram2.On("With", []string{"label2", "value2"}).Return(histogram2)
				histogram2.On("Observe", 0.250).Return()

				return map[string]kit.Histogram{
					"test_histogram_1": histogram1,
					"test_histogram_2": histogram2,
				}
			},
			events: []Event{
				{Name: "test_histogram_1", Labels: []string{"label1", "value1"}, Value: 0.125},
				{Name: "test_histogram_2", Labels: []string{"label2", "value2"}, Value: 0.250},
				{Name: "unknown_histogram", Labels: []string{"label3", "value3"}, Value: 0.500},
			},
			verifyMocks: true,
		},
		{
			name: "handle nil histogram gracefully",
			setupHistograms: func() map[string]kit.Histogram {
				return map[string]kit.Histogram{
					"test_histogram": nil,
				}
			},
			events: []Event{
				{Name: "test_histogram", Labels: []string{}, Value: 1.0},
			},
			shouldPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			histograms := tt.setupHistograms()
			observer := NewHistogramObserver(histograms)

			for _, event := range tt.events {
				if tt.shouldPanic {
					assert.NotPanics(t, func() {
						observer.HandleEvent(event)
					})
				} else {
					assert.NotPanics(t, func() {
						observer.HandleEvent(event)
					})
				}
			}

			if tt.verifyMocks {
				for _, histogram := range histograms {
					if mockHistogram, ok := histogram.(*MockHistogram); ok {
						mockHistogram.AssertExpectations(t)
					}
				}
			}
		})
	}
}
