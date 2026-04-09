// SPDX-FileCopyrightText: 2026 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"fmt"

	kit "github.com/go-kit/kit/metrics"
)

// Generic observers that can handle multiple metrics of the same type
type CounterObserver struct {
	counters map[string]kit.Counter
}

type GaugeObserver struct {
	gauges map[string]kit.Gauge
}

type HistogramObserver struct {
	histograms map[string]kit.Histogram
}

// NewCounterObserver creates a new observer for counter metrics
func NewCounterObserver(counters map[string]kit.Counter) *CounterObserver {
	return &CounterObserver{counters: counters}
}

// NewGaugeObserver creates a new observer for gauge metrics
func NewGaugeObserver(gauges map[string]kit.Gauge) *GaugeObserver {
	return &GaugeObserver{gauges: gauges}
}

// NewHistogramObserver creates a new observer for histogram metrics
func NewHistogramObserver(histograms map[string]kit.Histogram) *HistogramObserver {
	return &HistogramObserver{histograms: histograms}
}

func (c *CounterObserver) HandleEvent(event Event) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("ERROR: Prometheus panic for counter metric '%s': %v (labels: %v)\n", event.Name, r, event.Labels)
		}
	}()

	counter, ok := c.counters[event.Name]
	if !ok {
		// Silently ignore unknown metrics
		return
	}
	if counter == nil {
		fmt.Printf("ERROR: counter for metric '%s' is nil\n", event.Name)
		return
	}
	counter.With(event.Labels...).Add(event.Value)
}

func (g *GaugeObserver) HandleEvent(event Event) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("ERROR: Prometheus panic for gauge metric '%s': %v (labels: %v)\n", event.Name, r, event.Labels)
		}
	}()

	gauge, ok := g.gauges[event.Name]
	if !ok {
		// Silently ignore unknown metrics
		return
	}
	if gauge == nil {
		fmt.Printf("ERROR: gauge for metric '%s' is nil\n", event.Name)
		return
	}
	gauge.With(event.Labels...).Set(event.Value)
}

func (h *HistogramObserver) HandleEvent(event Event) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("ERROR: Prometheus panic for histogram metric '%s': %v (labels: %v)\n", event.Name, r, event.Labels)
		}
	}()

	histogram, ok := h.histograms[event.Name]
	if !ok {
		// Silently ignore unknown metrics
		return
	}
	if histogram == nil {
		fmt.Printf("ERROR: histogram for metric '%s' is nil\n", event.Name)
		return
	}
	histogram.With(event.Labels...).Observe(event.Value)
}
