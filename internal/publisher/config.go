// SPDX-FileCopyrightText: 2026 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package publisher

import (
	"time"

	"github.com/xmidt-org/wrpkafka"
)

// Config represents the YAML configuration for the WRP Kafka publisher.
// It can be unmarshaled via goschtalt and converted to functional options.
type Config struct {
	// Required fields
	Brokers []string

	// Topic routes for WRP message routing
	TopicRoutes []TopicRoute

	// Connection and retry settings
	RequestTimeout         time.Duration
	CleanupTimeout         time.Duration
	RequestRetries         int
	MaxBufferedRecords     int
	MaxBufferedBytes       int
	AllowAutoTopicCreation bool

	// SASL authentication
	SASL *SASLConfig

	// TLS configuration
	TLS *TLSConfig
}

// TopicRoute represents a WRP message routing configuration
type TopicRoute struct {
	Topic   string
	Pattern string
}

// ToWRPKafkaRoute converts a TopicRoute to a wrpkafka.TopicRoute
func (tr TopicRoute) ToWRPKafkaRoute() wrpkafka.TopicRoute {
	route := wrpkafka.TopicRoute{
		Topic:   tr.Topic,
		Pattern: wrpkafka.Pattern(tr.Pattern),
	}
	return route
}

// ToWRPKafkaRoutes converts all TopicRoutes to wrpkafka.TopicRoute slice
func (c Config) ToWRPKafkaRoutes() []wrpkafka.TopicRoute {
	routes := make([]wrpkafka.TopicRoute, len(c.TopicRoutes))
	for i, route := range c.TopicRoutes {
		routes[i] = route.ToWRPKafkaRoute()
	}
	return routes
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
