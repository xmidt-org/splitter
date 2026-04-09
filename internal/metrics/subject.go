// SPDX-FileCopyrightText: 2026 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"xmidt-org/splitter/internal/observe"

	kit "github.com/go-kit/kit/metrics"
)

type Metrics struct {
	ConsumerFetchErrors  kit.Counter
	ConsumerCommitErrors kit.Counter
	ConsumerPauses       kit.Gauge
	BucketKeyErrorCount  kit.Counter

	PublisherOutcomes      kit.Counter
	PublisherErrorsCounter kit.Counter

	// Kafka publisher metrics (wrpkafka event listeners)
	KafkaPublished      kit.Counter
	KafkaPublishLatency kit.Histogram
	Panics              kit.Counter
}

type Metric struct {
	counter   kit.Counter
	gauge     kit.Gauge
	histogram kit.Histogram
}

// New creates a new Subject for metric events with the 3 standard observers
func New(m Metrics) *observe.Subject[Event] {
	subject := observe.NewSubject[Event]()

	// Create counter observer with all counters
	counterMetrics := map[string]kit.Counter{
		"fetch_errors":                   m.ConsumerFetchErrors,
		"commit_errors":                  m.ConsumerCommitErrors,
		"bucket_key_error_count":         m.BucketKeyErrorCount,
		"publish_outcomes":               m.PublisherOutcomes,
		"publish_errors_total":           m.PublisherErrorsCounter,
		"kafka_messages_published_total": m.KafkaPublished,
		"panics_total":                   m.Panics,
	}

	// Create gauge observer with all gauges
	gaugeMetrics := map[string]kit.Gauge{
		"fetch_pauses": m.ConsumerPauses,
	}

	// Create histogram observer with all histograms
	histogramMetrics := map[string]kit.Histogram{
		"kafka_publish_latency_seconds": m.KafkaPublishLatency,
	}

	counterObserver := NewCounterObserver(counterMetrics)
	gaugeObserver := NewGaugeObserver(gaugeMetrics)
	histogramObserver := NewHistogramObserver(histogramMetrics)

	subject.Attach(counterObserver.HandleEvent)
	subject.Attach(gaugeObserver.HandleEvent)
	subject.Attach(histogramObserver.HandleEvent)

	return subject
}

func NewNoop() *observe.Subject[Event] {
	s := observe.NewSubject[Event]()
	// Don't attach any observers - events will be discarded
	return s
}
