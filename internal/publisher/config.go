// SPDX-FileCopyrightText: 2026 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package publisher

import (
	"fmt"
	"sort"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/xmidt-org/wrpkafka"
)

// Config represents the YAML configuration for the WRP Kafka publisher.
// It can be unmarshaled via goschtalt and converted to functional options.
type Config struct {
	// Required fields
	Brokers Brokers

	// Connection and retry settings
	RequestTimeout         time.Duration
	RecordDeliveryTimeout  time.Duration
	CleanupTimeout         time.Duration
	RequestRetries         int
	RecordRetries          int
	MaxBufferedRecords     int
	MaxBufferedBytes       int
	AllowAutoTopicCreation bool

	// DynamicConfig contains runtime-updatable Kafka producer settings
	// This includes all wrpkafka.DynamicConfig fields:
	//   - TopicMap: routing rules for WRP message routing
	//   - Headers: Kafka record headers
	//   - CompressionCodec: compression algorithm
	//   - Linger: batching delay
	//   - Acks: broker acknowledgment requirements
	DynamicConfig wrpkafka.DynamicConfig `yaml:"dynamic_config,omitempty"`

	// SASL authentication
	SASL *SASLConfig

	// TLS configuration
	TLS *TLSConfig

	// Prometheus metrics configuration
	Prometheus *PrometheusConfig
}

// Brokers describes the Kafka clusters the publisher can write to.
// Regions holds the broker list for every cluster (all regions); TargetRegion
// selects the preferred (primary) one.
type Brokers struct {
	RestartOnConfigChange bool
	TargetRegion          string
	Regions               map[string][]string

	// Failover configures automatic failover between the entries in Regions.
	Failover FailoverConfig
}

// FailoverConfig configures automatic regional failover. Each region gets its own
// circuit breaker (sony/gobreaker). While TargetRegion's breaker is open, traffic
// moves to the next region whose breaker is not open; once TargetRegion's breaker
// closes again, traffic moves back.
//
// Breakers are driven by asynchronous publish results, so a region is only seen as
// failing once records to it fail: with kgo.ErrRecordTimeout after
// RecordDeliveryTimeout, kgo.ErrRecordRetries after RecordRetries, or
// kgo.ErrMaxBuffered once the producer buffer fills. The Kafka client otherwise
// retries indefinitely, so set RecordDeliveryTimeout (or RecordRetries) to detect
// an unreachable region quickly. Records already buffered for a failed region are
// not moved to the new region.
//
// Production guidance (based on ~14k records/s, record_delivery_timeout: 30s,
// record_retries: 3, request_timeout: 30s, max_buffered_records: 20000):
//
//   - Regions must list distinct clusters. Both regions use the same SASL and TLS
//     settings, so the clusters must accept the same credentials.
//   - Breakers count individual records, so ConsecutiveFailures must be far above
//     the default of 5 at this volume (hundreds), or a brief slowdown that fills
//     the buffer could trigger a failover. HalfOpenSuccesses can be raised likewise.
//   - With a full buffer, ErrMaxBuffered fails every record immediately, so a dead
//     region is normally detected within seconds. Timeouts and retries are the
//     slower backup signal, and are not bounded by RecordDeliveryTimeout alone:
//     idempotent producing only enforces it between requests, and a black-holed
//     region can hold a produce request for ProduceRequestTimeout (10s, not
//     configurable through wrpkafka) plus RequestTimeout, so expect 40s or more.
//   - Probing a down region sends real traffic to it until its buffer fills, and
//     those records are lost (about 1.4s of traffic per probe). Use a long
//     OpenTimeout (minutes) to keep that loss infrequent.
//   - Failback is automatic: traffic returns to TargetRegion once its breaker
//     closes, and a restart always begins on TargetRegion.
//   - Each region has its own producer buffer, so worst-case buffered memory is
//     multiplied by the number of regions, and franz-go metrics gain a "region" label.
type FailoverConfig struct {
	// Enabled turns on regional failover. When false, only TargetRegion is used.
	Enabled bool

	// Regions is the ordered list of regions to fail over to, tried after
	// TargetRegion. When empty, every other region in Brokers.Regions is used in
	// alphabetical order.
	Regions []string

	// ConsecutiveFailures is the number of consecutive region-level publish failures
	// that opens a region's breaker. Default: 5.
	ConsecutiveFailures uint32

	// OpenTimeout is how long a region's breaker stays open before the region is
	// probed again. Default: 30s.
	OpenTimeout time.Duration

	// HalfOpenSuccesses is the number of consecutive successful publishes required
	// while probing to close the breaker. Default: 3.
	HalfOpenSuccesses uint32

	// Interval is the period after which failure counts are cleared while the
	// breaker is closed. Zero (default) never clears them.
	Interval time.Duration
}

const (
	defaultFailoverConsecutiveFailures = 5
	defaultFailoverOpenTimeout         = 30 * time.Second
	defaultFailoverHalfOpenSuccesses   = 3
)

// withDefaults returns a copy of the config with unset thresholds defaulted.
func (f FailoverConfig) withDefaults() FailoverConfig {
	if f.ConsecutiveFailures == 0 {
		f.ConsecutiveFailures = defaultFailoverConsecutiveFailures
	}
	if f.OpenTimeout <= 0 {
		f.OpenTimeout = defaultFailoverOpenTimeout
	}
	if f.HalfOpenSuccesses == 0 {
		f.HalfOpenSuccesses = defaultFailoverHalfOpenSuccesses
	}
	return f
}

// regionOrder returns the regions in priority order: TargetRegion first, then
// the failover regions. It returns just TargetRegion when failover is disabled.
func (b Brokers) regionOrder() []string {
	order := []string{b.TargetRegion}
	if !b.Failover.Enabled {
		return order
	}

	if len(b.Failover.Regions) > 0 {
		return append(order, b.Failover.Regions...)
	}

	others := make([]string, 0, len(b.Regions))
	for name := range b.Regions {
		if name != b.TargetRegion {
			others = append(others, name)
		}
	}
	sort.Strings(others)
	return append(order, others...)
}

// validateFailover checks that every failover region is a distinct, configured
// region with brokers, and that there is at least one region to fail over to.
func (b Brokers) validateFailover() error {
	if !b.Failover.Enabled {
		return nil
	}

	order := b.regionOrder()
	if len(order) < 2 {
		return ErrNoFailoverRegions
	}

	seen := make(map[string]struct{}, len(order))
	for _, name := range order {
		if _, dup := seen[name]; dup {
			return fmt.Errorf("failover region '%s' is listed more than once or equals target_region", name)
		}
		seen[name] = struct{}{}

		brokers, exists := b.Regions[name]
		if !exists {
			return fmt.Errorf("failover region '%s' not found in regions map", name)
		}
		if len(brokers) == 0 {
			return fmt.Errorf("failover region '%s' has no brokers configured", name)
		}
	}
	return nil
}

// SASLConfig represents SASL authentication configuration
type SASLConfig struct {
	Mechanism string `yaml:"mechanism"`
	Username  string `yaml:"username"`
	// #nosec G117
	Password string `yaml:"password"`
}

// TLSConfig represents TLS configuration
type TLSConfig struct {
	Enabled            bool   `yaml:"enabled"`
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify,omitempty"`
	CAFile             string `yaml:"ca_file,omitempty"`
	CertFile           string `yaml:"cert_file,omitempty"`
	KeyFile            string `yaml:"key_file,omitempty"`
}

// PrometheusConfig represents Prometheus metrics configuration for the publisher.
// This includes both franz-go metrics and application-level metrics.
type PrometheusConfig struct {
	// Namespace is the prometheus namespace for metrics (e.g., "xmidt")
	Namespace string `yaml:"namespace,omitempty"`

	// Subsystem is the prometheus subsystem name (e.g., "splitter")
	Subsystem string `yaml:"subsystem,omitempty"`

	// Registerer is the prometheus registerer to use for metrics.
	// If nil, metrics will be registered with the default prometheus registry.
	// This field is typically set programmatically, not via YAML.
	Registerer prometheus.Registerer `yaml:"-"`

	// Optional franz-go prometheus metrics options (disabled by default)
	// Application-level metrics (buffer utilization, publish counter, publish latency) are always enabled

	EnableBatchMetrics    bool `yaml:"enable_batch_metrics,omitempty"`
	EnableCompressedBytes bool `yaml:"enable_compressed_bytes,omitempty"`
	EnableGoCollectors    bool `yaml:"enable_go_collectors,omitempty"`
	WithClientLabel       bool `yaml:"with_client_label,omitempty"`
}
