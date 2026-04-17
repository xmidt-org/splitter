// SPDX-FileCopyrightText: 2026 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package consumer

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Config represents the YAML configuration for the Kafka consumer.
// It can be unmarshaled via goschtalt and converted to functional options.
type Config struct {
	// Required fields
	Brokers []string
	Topics  []string
	GroupID string

	// Session and heartbeat
	SessionTimeout    time.Duration
	HeartbeatInterval time.Duration
	RebalanceTimeout  time.Duration

	// Fetch configuration
	FetchMinBytes          int32
	FetchMaxBytes          int32
	FetchMaxWait           time.Duration
	FetchMaxPartitionBytes int32
	MaxConcurrentFetches   int
	// Auto-commit
	AutoCommitInterval time.Duration
	//DisableAutoCommit  bool          `yaml:"disable_auto_commit,omitempty"`

	// SASL authentication
	SASL *SASLConfig

	// TLS configuration
	TLS *TLSConfig

	// Retry and backoff
	RequestRetries int

	// Connection
	ConnIdleTimeout        time.Duration
	RequestTimeoutOverhead time.Duration
	// Client identification
	ClientID   string
	Rack       string
	InstanceID string
	// Offset management
	ConsumeFromBeginning bool

	// Fetch State Management
	ResumeDelaySeconds          int
	ConsecutiveFailureThreshold int

	// Prometheus metrics configuration
	Prometheus *PrometheusConfig
}

// SASLConfig contains SASL authentication configuration.
type SASLConfig struct {
	Mechanism string
	Username  string
	// #nosec G117 -- field required for configuration, not logged or exposed
	Password string
}

// TLSConfig contains TLS encryption configuration.
type TLSConfig struct {
	Enabled            bool
	CAFile             string
	CertFile           string
	KeyFile            string
	InsecureSkipVerify bool
}

// PrometheusConfig represents Prometheus metrics configuration for the consumer.
// This consolidates all prometheus/kprom settings for franz-go into a single struct.
type PrometheusConfig struct {
	// Namespace is the prometheus namespace for metrics (e.g., "xmidt")
	Namespace string `yaml:"namespace,omitempty"`

	// Subsystem is the prometheus subsystem name (e.g., "splitter_consumer")
	Subsystem string `yaml:"subsystem,omitempty"`

	// Registerer is the prometheus registerer to use for metrics.
	// If nil, metrics will be registered with the default prometheus registry.
	// This field is typically set programmatically, not via YAML.
	Registerer prometheus.Registerer `yaml:"-"`

	// Optional franz-go kprom metrics (disabled by default)
	// Core metrics (uncompressed bytes, records, batches, by_node, by_topic) are always enabled

	// EnableCompressedBytes tracks compressed bytes (fetch_compressed_bytes_total).
	// Default: false (adds overhead)
	EnableCompressedBytes bool `yaml:"enable_compressed_bytes,omitempty"`

	// EnableGoCollectors adds Go runtime metrics (goroutines, memory, etc.).
	// Default: false
	EnableGoCollectors bool `yaml:"enable_go_collectors,omitempty"`

	// WithClientLabel adds a "client_id" label to all metrics.
	// Default: false
	WithClientLabel bool `yaml:"with_client_label,omitempty"`
}

// IsCompressedBytesEnabled returns true if compressed bytes metric should be enabled.
func (p *PrometheusConfig) IsCompressedBytesEnabled() bool {
	return p.EnableCompressedBytes
}

// IsGoCollectorsEnabled returns true if Go runtime metrics should be enabled.
func (p *PrometheusConfig) IsGoCollectorsEnabled() bool {
	return p.EnableGoCollectors
}

// IsClientLabelEnabled returns true if client_id label should be added.
func (p *PrometheusConfig) IsClientLabelEnabled() bool {
	return p.WithClientLabel
}
