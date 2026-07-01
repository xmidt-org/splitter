// SPDX-FileCopyrightText: 2026 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package publisher

import (
	"log/slog"
	"os"
	"testing"
	"time"
	"xmidt-org/splitter/internal/log"
	"xmidt-org/splitter/internal/metrics"
	"xmidt-org/splitter/internal/observe"

	"github.com/stretchr/testify/suite"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/xmidt-org/wrpkafka"
)

// Test suite for Options (comprehensive option validation tests)
type OptionsValidationTestSuite struct {
	suite.Suite
}

// Test publisherConfig validation
func (suite *OptionsValidationTestSuite) TestPublisherConfig_ValidationRules() {
	tests := []struct {
		name        string
		config      *publisherConfig
		expectError bool
		expectedErr error
		description string
	}{
		{
			name: "valid_config",
			config: &publisherConfig{
				brokers: testBroker,
				topicRoutes: []wrpkafka.TopicRoute{
					{Topic: testTopicTest, Pattern: ".*"},
				},
			},
			expectError: false,
			description: "Should pass validation with valid config",
		},
		{
			name: "missing_brokers",
			config: &publisherConfig{
				brokers: Brokers{
					TargetRegion: testRegionUSEast1,
					Regions:      map[string][]string{},
				},
				topicRoutes: []wrpkafka.TopicRoute{
					{Topic: testTopicTest, Pattern: ".*"},
				},
			},
			expectError: true,
			expectedErr: ErrMissingBrokers,
			description: "Should fail validation when brokers are empty",
		},
		{
			name: "missing_topic_routes",
			config: &publisherConfig{
				brokers:     testBroker,
				topicRoutes: []wrpkafka.TopicRoute{},
			},
			expectError: true,
			expectedErr: ErrMissingTopicRoutes,
			description: "Should fail validation when topic routes are empty",
		},
		{
			name: "target_region_not_in_map",
			config: &publisherConfig{
				brokers: Brokers{
					TargetRegion: "nonexistent-region",
					Regions: map[string][]string{
						testRegionUSEast1: {testBrokerLocal},
					},
				},
				topicRoutes: []wrpkafka.TopicRoute{
					{Topic: testTopicTest, Pattern: ".*"},
				},
			},
			expectError: true,
			description: "Should fail validation when target_region not in regions map",
		},
		{
			name: "target_region_empty",
			config: &publisherConfig{
				brokers: Brokers{
					TargetRegion: "",
					Regions: map[string][]string{
						testRegionUSEast1: {testBrokerLocal},
					},
				},
				topicRoutes: []wrpkafka.TopicRoute{
					{Topic: testTopicTest, Pattern: ".*"},
				},
			},
			expectError: true,
			description: "Should fail validation when target_region is empty",
		},
		{
			name: "target_region_has_no_brokers",
			config: &publisherConfig{
				brokers: Brokers{
					TargetRegion: testRegionUSEast1,
					Regions: map[string][]string{
						testRegionUSEast1: {},
					},
				},
				topicRoutes: []wrpkafka.TopicRoute{
					{Topic: testTopicTest, Pattern: ".*"},
				},
			},
			expectError: true,
			description: "Should fail validation when target region has no brokers",
		},
		{
			name: "valid_multi_region_config",
			config: &publisherConfig{
				brokers: multiBroker,
				topicRoutes: []wrpkafka.TopicRoute{
					{Topic: testTopicTest, Pattern: ".*"},
				},
			},
			expectError: false,
			description: "Should pass validation with multi-region config",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			err := tt.config.validate()
			if tt.expectError {
				suite.Error(err, tt.description)
				if tt.expectedErr != nil {
					suite.ErrorIs(err, tt.expectedErr, tt.description)
				}
			} else {
				suite.NoError(err, tt.description)
			}
		})
	}
}

// Test WithBrokers option
func (suite *OptionsValidationTestSuite) TestWithBrokers() {
	p := &KafkaPublisher{config: &publisherConfig{}}
	opt := WithBrokers(testBroker)
	err := opt.apply(p)

	suite.NoError(err, "WithBrokers should not return error")
	suite.Equal(testBroker, p.config.brokers, "Should set brokers correctly")
}

// Test WithTopicRoutes option
func (suite *OptionsValidationTestSuite) TestWithTopicRoutes() {
	routes := []wrpkafka.TopicRoute{
		{Topic: testTopicEvents, Pattern: ".*"},
		{Topic: testTopicCommands, Pattern: "command:.*"},
	}

	p := &KafkaPublisher{config: &publisherConfig{}}
	opt := WithTopicRoutes(routes...)
	err := opt.apply(p)

	suite.NoError(err, "WithTopicRoutes should not return error")
	suite.Equal(routes, p.config.topicRoutes, "Should set topic routes correctly")
}

// Test WithLogger option
func (suite *OptionsValidationTestSuite) TestWithLogger() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	p := &KafkaPublisher{config: &publisherConfig{}}
	opt := WithLogger(logger)
	err := opt.apply(p)

	suite.NoError(err, "WithLogger should not return error")
	suite.Equal(logger, p.logger, "Should set logger correctly")
}

// Test WithLogger with nil logger
func (suite *OptionsValidationTestSuite) TestWithLogger_Nil() {
	p := &KafkaPublisher{config: &publisherConfig{}}
	opt := WithLogger(nil)
	err := opt.apply(p)

	suite.NoError(err, "WithLogger should not return error for nil logger")
	suite.Nil(p.logger, "Should not set logger when nil is passed")
}

// Test WithLogEmitter option
func (suite *OptionsValidationTestSuite) TestWithLogEmitter() {
	emitter := observe.NewSubject[log.Event]()

	p := &KafkaPublisher{config: &publisherConfig{}}
	opt := WithLogEmitter(emitter)
	err := opt.apply(p)

	suite.NoError(err, "WithLogEmitter should not return error")
	suite.Equal(emitter, p.logEmitter, "Should set log emitter correctly")
}

// Test WithLogEmitter with nil emitter
func (suite *OptionsValidationTestSuite) TestWithLogEmitter_Nil() {
	p := &KafkaPublisher{config: &publisherConfig{}}
	opt := WithLogEmitter(nil)
	err := opt.apply(p)

	suite.NoError(err, "WithLogEmitter should not return error for nil emitter")
	suite.Nil(p.logEmitter, "Should not set log emitter when nil is passed")
}

// Test WithMetricsEmitter option
func (suite *OptionsValidationTestSuite) TestWithMetricsEmitter() {
	emitter := observe.NewSubject[metrics.Event]()

	p := &KafkaPublisher{config: &publisherConfig{}}
	opt := WithMetricsEmitter(emitter)
	err := opt.apply(p)

	suite.NoError(err, "WithMetricsEmitter should not return error")
	suite.Equal(emitter, p.metricEmitter, "Should set metrics emitter correctly")
}

// Test WithMetricsEmitter with nil emitter
func (suite *OptionsValidationTestSuite) TestWithMetricsEmitter_Nil() {
	p := &KafkaPublisher{config: &publisherConfig{}}
	opt := WithMetricsEmitter(nil)
	err := opt.apply(p)

	suite.NoError(err, "WithMetricsEmitter should not return error for nil emitter")
	suite.Nil(p.metricEmitter, "Should not set metrics emitter when nil is passed")
}

// Test WithKafkaLogger option
func (suite *OptionsValidationTestSuite) TestWithKafkaLogger() {
	logger := &mockKgoLogger{}

	p := &KafkaPublisher{config: &publisherConfig{}}
	opt := WithKafkaLogger(logger)
	err := opt.apply(p)

	suite.NoError(err, "WithKafkaLogger should not return error")
	suite.Equal(logger, p.config.logger, "Should set kafka logger correctly")
}

// Test WithKafkaLogger with nil logger
func (suite *OptionsValidationTestSuite) TestWithKafkaLogger_Nil() {
	p := &KafkaPublisher{config: &publisherConfig{}}
	opt := WithKafkaLogger(nil)
	err := opt.apply(p)

	suite.NoError(err, "WithKafkaLogger should not return error for nil logger")
	suite.Nil(p.config.logger, "Should not set kafka logger when nil is passed")
}

// Test WithMaxBufferedRecords option
func (suite *OptionsValidationTestSuite) TestWithMaxBufferedRecords() {
	maxRecords := 5000

	p := &KafkaPublisher{config: &publisherConfig{}}
	opt := WithMaxBufferedRecords(maxRecords)
	err := opt.apply(p)

	suite.NoError(err, "WithMaxBufferedRecords should not return error")
	suite.Equal(maxRecords, p.config.maxBufferedRecords, "Should set max buffered records correctly")
}

// Test WithMaxBufferedBytes option
func (suite *OptionsValidationTestSuite) TestWithMaxBufferedBytes() {
	maxBytes := 1024 * 1024 * 10 // 10MB

	p := &KafkaPublisher{config: &publisherConfig{}}
	opt := WithMaxBufferedBytes(maxBytes)
	err := opt.apply(p)

	suite.NoError(err, "WithMaxBufferedBytes should not return error")
	suite.Equal(maxBytes, p.config.maxBufferedBytes, "Should set max buffered bytes correctly")
}

// Test WithRequestTimeout option
func (suite *OptionsValidationTestSuite) TestWithRequestTimeout() {
	timeout := 45 * time.Second

	p := &KafkaPublisher{config: &publisherConfig{}}
	opt := WithRequestTimeout(timeout)
	err := opt.apply(p)

	suite.NoError(err, "WithRequestTimeout should not return error")
	suite.Equal(timeout, p.config.requestTimeout, "Should set request timeout correctly")
}

// Test WithRecordDeliveryTimeout option
func (suite *OptionsValidationTestSuite) TestWithRecordDeliveryTimeout() {
	timeout := 2 * time.Minute

	p := &KafkaPublisher{config: &publisherConfig{}}
	opt := WithRecordDeliveryTimeout(timeout)
	err := opt.apply(p)

	suite.NoError(err, "WithRecordDeliveryTimeout should not return error")
	suite.Equal(timeout, p.config.recordDeliveryTimeout, "Should set record delivery timeout correctly")
}

// Test WithCleanupTimeout option
func (suite *OptionsValidationTestSuite) TestWithCleanupTimeout() {
	timeout := 90 * time.Second

	p := &KafkaPublisher{config: &publisherConfig{}}
	opt := WithCleanupTimeout(timeout)
	err := opt.apply(p)

	suite.NoError(err, "WithCleanupTimeout should not return error")
	suite.Equal(timeout, p.config.cleanupTimeout, "Should set cleanup timeout correctly")
}

// Test WithMaxRequestRetries option
func (suite *OptionsValidationTestSuite) TestWithMaxRequestRetries() {
	retries := 5

	p := &KafkaPublisher{config: &publisherConfig{}}
	opt := WithMaxRequestRetries(retries)
	err := opt.apply(p)

	suite.NoError(err, "WithMaxRequestRetries should not return error")
	suite.Equal(retries, p.config.maxRequestRetries, "Should set max request retries correctly")
}

// Test WithMaxRecordRetries option
func (suite *OptionsValidationTestSuite) TestWithMaxRecordRetries() {
	retries := 10

	p := &KafkaPublisher{config: &publisherConfig{}}
	opt := WithMaxRecordRetries(retries)
	err := opt.apply(p)

	suite.NoError(err, "WithMaxRecordRetries should not return error")
	suite.Equal(retries, p.config.maxRecordRetries, "Should set max record retries correctly")
}

// Test WithAllowAutoTopicCreation option
func (suite *OptionsValidationTestSuite) TestWithAllowAutoTopicCreation() {
	tests := []struct {
		name  string
		allow bool
	}{
		{name: "enabled", allow: true},
		{name: "disabled", allow: false},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			p := &KafkaPublisher{config: &publisherConfig{}}
			opt := WithAllowAutoTopicCreation(tt.allow)
			err := opt.apply(p)

			suite.NoError(err, "WithAllowAutoTopicCreation should not return error")
			suite.Equal(tt.allow, p.config.allowAutoTopicCreation, "Should set allow auto topic creation correctly")
		})
	}
}

// Test WithSASLConfig option
func (suite *OptionsValidationTestSuite) TestWithSASLConfig() {
	tests := []struct {
		name        string
		config      *SASLConfig
		expectError bool
		description string
	}{
		{
			name: "valid_plain",
			config: &SASLConfig{
				Mechanism: saslMechanismPlain,
				Username:  testUsername,
				Password:  testPassword,
			},
			expectError: false,
			description: "Should configure PLAIN SASL correctly",
		},
		{
			name: "valid_scram256",
			config: &SASLConfig{
				Mechanism: saslMechanismScram256,
				Username:  testUsername,
				Password:  testPassword,
			},
			expectError: false,
			description: "Should configure SCRAM-SHA-256 correctly",
		},
		{
			name: "valid_scram512",
			config: &SASLConfig{
				Mechanism: saslMechanismScram512,
				Username:  testUsername,
				Password:  testPassword,
			},
			expectError: false,
			description: "Should configure SCRAM-SHA-512 correctly",
		},
		{
			name:        "nil_config",
			config:      nil,
			expectError: false,
			description: "Should handle nil SASL config gracefully",
		},
		{
			name: "missing_username",
			config: &SASLConfig{
				Mechanism: saslMechanismPlain,
				Username:  "",
				Password:  testPassword,
			},
			expectError: true,
			description: "Should fail when username is missing",
		},
		{
			name: "missing_password",
			config: &SASLConfig{
				Mechanism: saslMechanismPlain,
				Username:  testUsername,
				Password:  "",
			},
			expectError: true,
			description: "Should fail when password is missing",
		},
		{
			name: "unsupported_mechanism",
			config: &SASLConfig{
				Mechanism: "OAUTH",
				Username:  testUsername,
				Password:  testPassword,
			},
			expectError: true,
			description: "Should fail with unsupported mechanism",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			p := &KafkaPublisher{config: &publisherConfig{}}
			opt := WithSASLConfig(tt.config)
			err := opt.apply(p)

			if tt.expectError {
				suite.Error(err, tt.description)
			} else {
				suite.NoError(err, tt.description)
				if tt.config != nil && tt.config.Username != "" && tt.config.Password != "" {
					suite.NotNil(p.config.sasl, "Should set SASL mechanism")
				}
			}
		})
	}
}

// Test WithTLSConfig option
func (suite *OptionsValidationTestSuite) TestWithTLSConfig() {
	tests := []struct {
		name        string
		config      *TLSConfig
		expectError bool
		description string
	}{
		{
			name:        "nil_config",
			config:      nil,
			expectError: false,
			description: "Should handle nil TLS config gracefully",
		},
		{
			name: "disabled_tls",
			config: &TLSConfig{
				Enabled: false,
			},
			expectError: false,
			description: "Should handle disabled TLS gracefully",
		},
		{
			name: "enabled_with_insecure_skip_verify",
			config: &TLSConfig{
				Enabled:            true,
				InsecureSkipVerify: true,
			},
			expectError: false,
			description: "Should enable TLS with insecure skip verify",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			p := &KafkaPublisher{config: &publisherConfig{}}
			opt := WithTLSConfig(tt.config)
			err := opt.apply(p)

			if tt.expectError {
				suite.Error(err, tt.description)
			} else {
				suite.NoError(err, tt.description)
				if tt.config != nil && tt.config.Enabled {
					suite.NotNil(p.config.tls, "Should set TLS config")
				}
			}
		})
	}
}

// Test WithPublishEventListener option
func (suite *OptionsValidationTestSuite) TestWithPublishEventListener() {
	callCount := 0
	listener := func(event *wrpkafka.PublishEvent) {
		callCount++
	}

	p := &KafkaPublisher{config: &publisherConfig{}}
	opt := WithPublishEventListener(listener)
	err := opt.apply(p)

	suite.NoError(err, "WithPublishEventListener should not return error")
	suite.Len(p.config.publishEventListeners, 1, "Should add listener to list")
}

// Test WithPublishEventListener with nil listener
func (suite *OptionsValidationTestSuite) TestWithPublishEventListener_Nil() {
	p := &KafkaPublisher{config: &publisherConfig{}}
	opt := WithPublishEventListener(nil)
	err := opt.apply(p)

	suite.NoError(err, "WithPublishEventListener should not return error for nil listener")
	suite.Len(p.config.publishEventListeners, 0, "Should not add nil listener")
}

// Test WithSASLPlain convenience method
func (suite *OptionsValidationTestSuite) TestWithSASLPlain() {
	p := &KafkaPublisher{config: &publisherConfig{}}
	opt := WithSASLPlain(testUsername, testPassword)
	err := opt.apply(p)

	suite.NoError(err, "WithSASLPlain should not return error")
	suite.NotNil(p.config.sasl, "Should set SASL mechanism")
}

// Test WithSASLScram256 convenience method
func (suite *OptionsValidationTestSuite) TestWithSASLScram256() {
	p := &KafkaPublisher{config: &publisherConfig{}}
	opt := WithSASLScram256(testUsername, testPassword)
	err := opt.apply(p)

	suite.NoError(err, "WithSASLScram256 should not return error")
	suite.NotNil(p.config.sasl, "Should set SASL mechanism")
}

// Test WithSASLScram512 convenience method
func (suite *OptionsValidationTestSuite) TestWithSASLScram512() {
	p := &KafkaPublisher{config: &publisherConfig{}}
	opt := WithSASLScram512(testUsername, testPassword)
	err := opt.apply(p)

	suite.NoError(err, "WithSASLScram512 should not return error")
	suite.NotNil(p.config.sasl, "Should set SASL mechanism")
}

// Test WithTLS convenience method
func (suite *OptionsValidationTestSuite) TestWithTLS() {
	p := &KafkaPublisher{config: &publisherConfig{}}
	opt := WithTLS()
	err := opt.apply(p)

	suite.NoError(err, "WithTLS should not return error")
	suite.NotNil(p.config.tls, "Should set TLS config")
}

// Test WithPrometheusConfig option
func (suite *OptionsValidationTestSuite) TestWithPrometheusConfig() {
	config := &PrometheusConfig{
		Namespace: testNamespaceXmidt,
		Subsystem: testSubsystemSplitter,
	}

	p := &KafkaPublisher{config: &publisherConfig{}}
	opt := WithPrometheusConfig(config)
	err := opt.apply(p)

	suite.NoError(err, "WithPrometheusConfig should not return error")
	suite.Equal(config, p.config.prometheus, "Should set Prometheus config correctly")
}

// Test WithPrometheusConfig with nil config
func (suite *OptionsValidationTestSuite) TestWithPrometheusConfig_Nil() {
	p := &KafkaPublisher{config: &publisherConfig{}}
	opt := WithPrometheusConfig(nil)
	err := opt.apply(p)

	suite.NoError(err, "WithPrometheusConfig should not return error for nil config")
	suite.Nil(p.config.prometheus, "Should not set Prometheus config when nil is passed")
}

// Test WithDynamicConfig option
func (suite *OptionsValidationTestSuite) TestWithDynamicConfig() {
	dynamicConfig := wrpkafka.DynamicConfig{
		TopicMap: []wrpkafka.TopicRoute{
			{Topic: testTopicEvents, Pattern: ".*"},
		},
		Headers: map[string][]string{
			"x-source": {"splitter"},
		},
		CompressionCodec: wrpkafka.CompressionSnappy,
		Linger:           100 * time.Millisecond,
		Acks:             wrpkafka.AcksAll,
	}

	p := &KafkaPublisher{config: &publisherConfig{}}
	opt := WithDynamicConfig(dynamicConfig)
	err := opt.apply(p)

	suite.NoError(err, "WithDynamicConfig should not return error")
	suite.Equal(dynamicConfig, p.config.dynamicConfig, "Should set dynamic config correctly")
}

// Test WithHeaders option
func (suite *OptionsValidationTestSuite) TestWithHeaders() {
	headers := map[string][]string{
		"x-source":      {"splitter"},
		"x-environment": {"prod", "staging"},
	}

	p := &KafkaPublisher{config: &publisherConfig{}}
	opt := WithHeaders(headers)
	err := opt.apply(p)

	suite.NoError(err, "WithHeaders should not return error")
	suite.Equal(headers, p.config.dynamicConfig.Headers, "Should set headers correctly")
}

// Test WithHeaders with nil headers
func (suite *OptionsValidationTestSuite) TestWithHeaders_Nil() {
	p := &KafkaPublisher{config: &publisherConfig{}}
	opt := WithHeaders(nil)
	err := opt.apply(p)

	suite.NoError(err, "WithHeaders should not return error for nil headers")
	suite.Nil(p.config.dynamicConfig.Headers, "Should not set headers when nil is passed")
}

// Test WithCompressionCodec option
func (suite *OptionsValidationTestSuite) TestWithCompressionCodec() {
	tests := []struct {
		name  string
		codec wrpkafka.Compression
	}{
		{name: "none", codec: wrpkafka.CompressionNone},
		{name: "gzip", codec: wrpkafka.CompressionGzip},
		{name: "snappy", codec: wrpkafka.CompressionSnappy},
		{name: "lz4", codec: wrpkafka.CompressionLz4},
		{name: "zstd", codec: wrpkafka.CompressionZstd},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			p := &KafkaPublisher{config: &publisherConfig{}}
			opt := WithCompressionCodec(tt.codec)
			err := opt.apply(p)

			suite.NoError(err, "WithCompressionCodec should not return error")
			suite.Equal(tt.codec, p.config.dynamicConfig.CompressionCodec, "Should set compression codec correctly")
		})
	}
}

// Test WithLinger option
func (suite *OptionsValidationTestSuite) TestWithLinger() {
	tests := []struct {
		name   string
		linger time.Duration
	}{
		{name: "zero", linger: 0},
		{name: "positive", linger: 100 * time.Millisecond},
		{name: "negative", linger: -1 * time.Millisecond},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			p := &KafkaPublisher{config: &publisherConfig{}}
			opt := WithLinger(tt.linger)
			err := opt.apply(p)

			suite.NoError(err, "WithLinger should not return error")
			suite.Equal(tt.linger, p.config.dynamicConfig.Linger, "Should set linger correctly")
		})
	}
}

// Test WithAcks option
func (suite *OptionsValidationTestSuite) TestWithAcks() {
	tests := []struct {
		name string
		acks wrpkafka.Acks
	}{
		{name: "none", acks: wrpkafka.AcksNone},
		{name: "leader", acks: wrpkafka.AcksLeader},
		{name: "all", acks: wrpkafka.AcksAll},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			p := &KafkaPublisher{config: &publisherConfig{}}
			opt := WithAcks(tt.acks)
			err := opt.apply(p)

			suite.NoError(err, "WithAcks should not return error")
			suite.Equal(tt.acks, p.config.dynamicConfig.Acks, "Should set acks correctly")
		})
	}
}

// Run the options validation test suite
func TestOptionsValidationTestSuite(t *testing.T) {
	suite.Run(t, new(OptionsValidationTestSuite))
}

// Mock kgo.Logger for testing
type mockKgoLogger struct{}

func (m *mockKgoLogger) Level() kgo.LogLevel {
	return kgo.LogLevelInfo
}

func (m *mockKgoLogger) Log(level kgo.LogLevel, msg string, keyvals ...any) {
	// No-op for testing
}
