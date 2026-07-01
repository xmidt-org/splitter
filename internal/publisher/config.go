// SPDX-FileCopyrightText: 2026 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package publisher

import (
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

type Brokers struct {
	RestartOnConfigChange bool
	TargetRegion          string
	Regions               map[string][]string
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
