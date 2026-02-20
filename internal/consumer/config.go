// SPDX-FileCopyrightText: 2026 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package consumer

import (
	"time"
)

// Config represents the YAML configuration for the Kafka consumer.
// It can be unmarshaled via goschtalt and converted to functional options.
type Config struct {
	// Required fields
	Brokers []string `yaml:"brokers"`
	Topics  []string `yaml:"topics"`
	GroupID string   `yaml:"group_id"`

	// Session and heartbeat
	SessionTimeout    time.Duration `yaml:"session_timeout,omitempty"`
	HeartbeatInterval time.Duration `yaml:"heartbeat_interval,omitempty"`
	RebalanceTimeout  time.Duration `yaml:"rebalance_timeout,omitempty"`

	// Fetch configuration
	FetchMinBytes          int32         `yaml:"fetch_min_bytes,omitempty"`
	FetchMaxBytes          int32         `yaml:"fetch_max_bytes,omitempty"`
	FetchMaxWait           time.Duration `yaml:"fetch_max_wait,omitempty"`
	FetchMaxPartitionBytes int32         `yaml:"fetch_max_partition_bytes,omitempty"`
	MaxConcurrentFetches   int           `yaml:"max_concurrent_fetches,omitempty"`
	// Auto-commit
	AutoCommitInterval time.Duration `yaml:"auto_commit_interval,omitempty"`
	//DisableAutoCommit  bool          `yaml:"disable_auto_commit,omitempty"`

	// SASL authentication
	SASL *SASLConfig `yaml:"sasl,omitempty"`

	// TLS configuration
	TLS *TLSConfig `yaml:"tls,omitempty"`

	// Retry and backoff
	RequestRetries int `yaml:"request_retries,omitempty"`

	// Connection
	ConnIdleTimeout        time.Duration `yaml:"conn_idle_timeout,omitempty"`
	RequestTimeoutOverhead time.Duration `yaml:"request_timeout_overhead,omitempty"`

	// Client identification
	ClientID   string `yaml:"client_id,omitempty"`
	Rack       string `yaml:"rack,omitempty"`
	InstanceID string `yaml:"instance_id,omitempty"`

	// Offset management
	ConsumeFromBeginning bool `yaml:"consume_from_beginning,omitempty"`

	// Fetch State Management
	ResumeDelaySeconds          int `yaml:"resume_delay_seconds,omitempty"`
	ConsecutiveFailureThreshold int `yaml:"consecutive_failure_threshold,omitempty"`
}

// SASLConfig contains SASL authentication configuration.
type SASLConfig struct {
	Mechanism string `yaml:"mechanism"` // PLAIN, SCRAM-SHA-256, SCRAM-SHA-512
	Username  string `yaml:"username"`
	Password  string `yaml:"password"`
}

// TLSConfig contains TLS encryption configuration.
type TLSConfig struct {
	Enabled            bool   `yaml:"enabled"`
	CAFile             string `yaml:"ca_file,omitempty"`
	CertFile           string `yaml:"cert_file,omitempty"`
	KeyFile            string `yaml:"key_file,omitempty"`
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify,omitempty"`
}

// Duration is a wrapper around time.Duration that supports YAML unmarshaling.
// type Duration time.Duration

// // UnmarshalText implements encoding.TextUnmarshaler for Duration.
// func (d *Duration) UnmarshalText(text []byte) error {
// 	duration, err := time.ParseDuration(string(text))
// 	if err != nil {
// 		return err
// 	}
// 	*d = Duration(duration)
// 	return nil
// }

// // MarshalText implements encoding.TextMarshaler for Duration.
// func (d Duration) MarshalText() ([]byte, error) {
// 	return []byte(time.Duration(d).String()), nil
// }

// // Validate checks that required fields are set and values are valid.
// func (c *Config) Validate() error {
// 	if len(c.Brokers) == 0 {
// 		return errors.New("brokers are required")
// 	}
// 	if len(c.Topics) == 0 {
// 		return errors.New("topics are required")
// 	}
// 	if c.GroupID == "" {
// 		return errors.New("group_id is required")
// 	}

// 	// Validate SASL if provided
// 	if c.SASL != nil {
// 		if err := c.SASL.Validate(); err != nil {
// 			return fmt.Errorf("invalid sasl config: %w", err)
// 		}
// 	}

// 	// Validate TLS if provided
// 	if c.TLS != nil && c.TLS.Enabled {
// 		if err := c.TLS.Validate(); err != nil {
// 			return fmt.Errorf("invalid tls config: %w", err)
// 		}
// 	}

// 	return nil
// }

// // Validate checks SASL configuration.
// func (s *SASLConfig) Validate() error {
// 	if s == nil {
// 		return nil
// 	}

// 	switch s.Mechanism {
// 	case "PLAIN", "SCRAM-SHA-256", "SCRAM-SHA-512":
// 		// Valid mechanisms
// 	default:
// 		return fmt.Errorf("unsupported sasl mechanism: %s", s.Mechanism)
// 	}

// 	if s.Username == "" {
// 		return errors.New("sasl username is required")
// 	}
// 	if s.Password == "" {
// 		return errors.New("sasl password is required")
// 	}

// 	return nil
// }

// // Validate checks TLS configuration.
// func (t *TLSConfig) Validate() error {
// 	if t == nil || !t.Enabled {
// 		return nil
// 	}

// 	// If CA file is specified, it must exist
// 	if t.CAFile != "" {
// 		if _, err := os.Stat(t.CAFile); err != nil {
// 			return fmt.Errorf("ca_file not found: %w", err)
// 		}
// 	}

// 	// If cert and key are specified, both must exist
// 	if t.CertFile != "" || t.KeyFile != "" {
// 		if t.CertFile == "" || t.KeyFile == "" {
// 			return errors.New("both cert_file and key_file must be specified together")
// 		}
// 		if _, err := os.Stat(t.CertFile); err != nil {
// 			return fmt.Errorf("cert_file not found: %w", err)
// 		}
// 		if _, err := os.Stat(t.KeyFile); err != nil {
// 			return fmt.Errorf("key_file not found: %w", err)
// 		}
// 	}

// 	return nil
// }

// // ToTLSConfig converts TLSConfig to *tls.Config.
// func (t *TLSConfig) ToTLSConfig() (*tls.Config, error) {
// 	if !t.Enabled {
// 		return nil, nil
// 	}

// 	tlsConfig := &tls.Config{
// 		InsecureSkipVerify: t.InsecureSkipVerify,
// 	}

// 	// Load CA certificate if provided
// 	if t.CAFile != "" {
// 		caCert, err := os.ReadFile(t.CAFile)
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to read ca file: %w", err)
// 		}

// 		caCertPool := x509.NewCertPool()
// 		if !caCertPool.AppendCertsFromPEM(caCert) {
// 			return nil, errors.New("failed to parse ca certificate")
// 		}
// 		tlsConfig.RootCAs = caCertPool
// 	}

// 	// Load client certificate if provided
// 	if t.CertFile != "" && t.KeyFile != "" {
// 		cert, err := tls.LoadX509KeyPair(t.CertFile, t.KeyFile)
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to load client certificate: %w", err)
// 		}
// 		tlsConfig.Certificates = []tls.Certificate{cert}
// 	}

// 	return tlsConfig, nil
// }
