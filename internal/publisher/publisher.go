// SPDX-FileCopyrightText: 2026 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package publisher

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"xmidt-org/splitter/internal/log"
	"xmidt-org/splitter/internal/metrics"
	"xmidt-org/splitter/internal/observe"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/xmidt-org/wrp-go/v5"
	"github.com/xmidt-org/wrpkafka"
)

// Unused constant - kept for potential future use
// const (
// 	errorTypeNone = "none" //nolint:unused
// )

var (
	ErrPublisherNotStarted     = errors.New("publisher is not started")
	ErrPublisherAlreadyStarted = errors.New("publisher is already started")
)

// Interface for dependency injection and testing
type Publisher interface {
	Produce(ctx context.Context, msg *wrp.Message) (wrpkafka.Outcome, error)
	Start() error
	Stop(ctx context.Context) error
}

// Publisher implements the publisher using wrpkafka.
// It manages the publisher lifecycle, message production, and graceful shutdown.
type KafkaPublisher struct {
	// wrpPublisher is the publisher for the target (primary) region.
	wrpPublisher  *wrpkafka.Publisher
	failover      *failoverGroup
	config        *publisherConfig
	logger        *slog.Logger
	logEmitter    *observe.Subject[log.Event]
	metricEmitter *observe.Subject[metrics.Event]

	mu      sync.RWMutex
	started bool
}

// New creates a new KafkaPublisher with the provided options.
// Required options: WithBrokers, WithTopicRoutes
func New(opts ...Option) (*KafkaPublisher, error) {
	// Create publisher with initial config
	publisher := &KafkaPublisher{
		config:        &publisherConfig{},
		logger:        slog.Default(),
		logEmitter:    log.NewNoop(),
		metricEmitter: metrics.NewNoop(),
	}

	// Apply all provided options
	for _, opt := range opts {
		if err := opt.apply(publisher); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	// Validate required options
	if err := publisher.config.validate(); err != nil {
		return nil, err
	}

	// Create one wrpkafka publisher per region, target region first. Without
	// failover there is only the target region and no circuit breaker.
	var failoverCfg *FailoverConfig
	if publisher.config.brokers.Failover.Enabled {
		cfg := publisher.config.brokers.Failover.withDefaults()
		failoverCfg = &cfg
	}

	publisher.failover = newFailoverGroup(publisher.logEmitter, publisher.metricEmitter)
	for i, name := range publisher.config.brokers.regionOrder() {
		wrpPublisher := publisher.newWRPPublisher(name)
		if i == 0 {
			publisher.wrpPublisher = wrpPublisher
		}
		publisher.failover.add(name, wrpPublisher, failoverCfg)
	}

	return publisher, nil
}

// newWRPPublisher creates the underlying wrpkafka publisher for a region.
func (p *KafkaPublisher) newWRPPublisher(regionName string) *wrpkafka.Publisher {
	// Build dynamic config with TopicMap from converted routes
	dynamicConfig := p.config.dynamicConfig
	dynamicConfig.TopicMap = p.config.topicRoutes

	return &wrpkafka.Publisher{
		Brokers:                      p.config.brokers.Regions[regionName],
		InitialDynamicConfig:         dynamicConfig,
		SASL:                         p.config.sasl,
		TLS:                          p.config.tls,
		MaxBufferedRecords:           p.config.maxBufferedRecords,
		MaxBufferedBytes:             p.config.maxBufferedBytes,
		RequestTimeout:               p.config.requestTimeout,
		RecordDeliveryTimeout:        p.config.recordDeliveryTimeout,
		CleanupTimeout:               p.config.cleanupTimeout,
		MaxRequestRetries:            p.config.maxRequestRetries,
		MaxRecordRetries:             p.config.maxRecordRetries,
		AllowAutoTopicCreation:       p.config.allowAutoTopicCreation,
		Logger:                       p.config.logger,
		InitialPublishEventListeners: p.config.publishEventListeners,
		Prometheus:                   p.regionPrometheusConfig(regionName),
	}
}

// regionPrometheusConfig returns the franz-go metrics config for a region. When
// failover is enabled every region registers the same metrics, so each region's
// registerer adds a "region" label to keep the registrations distinct.
func (p *KafkaPublisher) regionPrometheusConfig(regionName string) wrpkafka.PrometheusConfig {
	cfg := p.toWRPKafkaPrometheusConfig()
	if !p.config.brokers.Failover.Enabled || cfg.Namespace == "" {
		return cfg
	}

	registerer := cfg.Registerer
	if registerer == nil {
		registerer = prometheus.DefaultRegisterer
	}
	cfg.Registerer = prometheus.WrapRegistererWith(prometheus.Labels{metrics.RegionLabel: regionName}, registerer)
	return cfg
}

// toWRPKafkaPrometheusConfig converts our PrometheusConfig to wrpkafka.PrometheusConfig.
func (p *KafkaPublisher) toWRPKafkaPrometheusConfig() wrpkafka.PrometheusConfig {
	if p.config.prometheus == nil {
		return wrpkafka.PrometheusConfig{}
	}

	cfg := wrpkafka.PrometheusConfig{
		Namespace:             p.config.prometheus.Namespace,
		Subsystem:             p.config.prometheus.Subsystem,
		Registerer:            p.config.prometheus.Registerer,
		EnableBatchMetrics:    p.config.prometheus.EnableBatchMetrics,
		EnableCompressedBytes: p.config.prometheus.EnableCompressedBytes,
		EnableGoCollectors:    p.config.prometheus.EnableGoCollectors,
		WithClientLabel:       p.config.prometheus.WithClientLabel,
	}

	return cfg
}

// Start initializes and starts the publisher.
// This must be called before producing messages.
func (p *KafkaPublisher) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.started {
		return ErrPublisherAlreadyStarted
	}

	// log the target broker at startup for visibility
	p.logEmitter.Notify(log.NewEvent(log.LevelInfo, "Starting WRP publisher with configured target broker", map[string]any{
		"brokers":  p.config.brokers.TargetRegion,
		"failover": p.config.brokers.Failover.Enabled,
	}))

	if p.config.brokers.Failover.Enabled && p.config.recordDeliveryTimeout <= 0 && p.config.maxRecordRetries <= 0 {
		p.logEmitter.Notify(log.NewEvent(log.LevelWarn, "failover is enabled without record_delivery_timeout or record_retries; "+
			"an unreachable region will only be detected once the producer buffer is full", nil))
	}

	// Listener for all publish events (success and failure), shared by every region.
	// Application-level metrics are always enabled
	publishListener := func(event *wrpkafka.PublishEvent) {

		errorType := "success"
		if event.Error != nil {
			errorType = event.ErrorType
		}

		labels := []string{
			metrics.TopicLabel, event.Topic,
			metrics.TopicShardStrategyLabel, event.TopicShardStrategy,
			metrics.ErrorTypeLabel, errorType,
		}

		p.metricEmitter.Notify(metrics.Event{
			Name:   metrics.KafkaPublished,
			Value:  1,
			Labels: labels,
		})

		// Record latency for all events
		p.metricEmitter.Notify(metrics.Event{
			Name:  metrics.KafkaPublishLatency,
			Value: event.Duration.Seconds(),
			Labels: []string{
				metrics.TopicLabel, event.Topic,
				metrics.ErrorTypeLabel, func() string {
					if event.Error != nil {
						return event.ErrorType
					}
					return "none"
				}(),
			},
		})

	}

	// Set up buffer utilization function for automatic Prometheus scraping (always enabled)
	if p.config.maxBufferedRecords > 0 {
		metrics.BufferUtilization = p.failover.bufferedRecords
	}

	// Start the wrpkafka publisher(s)
	if err := p.failover.start(publishListener); err != nil {
		p.logEmitter.Notify(log.NewEvent(log.LevelError, "Failed to start WRP publisher", map[string]any{
			"error": err.Error(),
		}))
		return fmt.Errorf("failed to start wrpkafka publisher: %w", err)
	}

	p.started = true
	p.logEmitter.Notify(log.NewEvent(log.LevelInfo, "WRP publisher started successfully", nil))

	return nil
}

// Stop shuts down the publisher.
func (p *KafkaPublisher) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.started {
		return nil
	}

	// Clear the buffer utilization function
	if p.config.maxBufferedRecords > 0 {
		metrics.BufferUtilization = nil
	}

	p.failover.stop(ctx)
	p.started = false
	p.logEmitter.Notify(log.NewEvent(log.LevelInfo, "WRP publisher stopped", nil))

	return nil
}

func (p *KafkaPublisher) IsStarted() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.started
}

// Produce sends a WRP message to the appropriate Kafka topic based on routing rules.
// Returns the outcome of the produce operation and any error encountered.
func (p *KafkaPublisher) Produce(ctx context.Context, msg *wrp.Message) (wrpkafka.Outcome, error) {
	p.mu.RLock()
	started := p.started
	p.mu.RUnlock()

	if !started {
		p.logEmitter.Notify(log.NewEvent(log.LevelError, "publisher not started", nil))
		p.metricEmitter.Notify(metrics.Event{
			Name:  metrics.PublisherErrorsCounter,
			Value: 1,
			Labels: []string{
				metrics.ErrorTypeLabel, "not_started",
			},
		})
		return wrpkafka.Failed, ErrPublisherNotStarted
	}

	outcome, err := p.failover.produce(ctx, msg)
	if err != nil {
		p.logEmitter.Notify(log.NewEvent(log.LevelError, "Failed to produce WRP message", map[string]any{
			"error":        err.Error(),
			"message_type": msg.Type.String(),
			"source":       msg.Source,
			"destination":  msg.Destination,
		}))
		return outcome, fmt.Errorf("failed to produce message: %w", err)
	}

	p.logEmitter.Notify(log.NewEvent(log.LevelDebug, "WRP message produced successfully", map[string]any{
		"message_type": msg.Type.String(),
		"source":       msg.Source,
		"destination":  msg.Destination,
		"outcome":      outcome.String(),
	}))

	return outcome, nil
}
