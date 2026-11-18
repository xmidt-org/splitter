// SPDX-FileCopyrightText: 2026 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package publisher

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync/atomic"
	"time"

	"xmidt-org/splitter/internal/log"
	"xmidt-org/splitter/internal/metrics"
	"xmidt-org/splitter/internal/observe"

	"github.com/sony/gobreaker/v2"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/xmidt-org/wrp-go/v5"
	"github.com/xmidt-org/wrpkafka"
)

// ErrNoRegionAvailable is returned when every region's circuit breaker is open.
var ErrNoRegionAvailable = errors.New("no kafka region available")

// producer is the subset of *wrpkafka.Publisher used by the failover group.
type producer interface {
	Produce(ctx context.Context, msg *wrp.Message) (wrpkafka.Outcome, error)
	Start() error
	Stop(ctx context.Context)
	AddPublishEventListener(fn func(*wrpkafka.PublishEvent)) func()
	BufferedRecords() (currentRecords, maxRecords int, currentBytes, maxBytes int64)
}

// region is a single Kafka cluster and, when failover is enabled, its breaker.
type region struct {
	name     string
	producer producer
	breaker  *gobreaker.CircuitBreaker[struct{}]
	detach   []func()
}

// isOpen reports whether the region should currently be skipped. Calling State
// also moves an expired open breaker to half-open so the region gets probed.
func (r *region) isOpen() bool {
	return r.breaker != nil && r.breaker.State() == gobreaker.StateOpen
}

// record feeds a publish result into the region's breaker. Errors that do not
// indicate an unhealthy region are ignored rather than counted either way.
func (r *region) record(event *wrpkafka.PublishEvent) {
	if event.Error != nil && !isRegionFailure(event.Error) {
		return
	}
	// Execute is only used to record the result; it ignores results while open.
	_, _ = r.breaker.Execute(func() (struct{}, error) {
		return struct{}{}, event.Error
	})
}

// regionFailureErrors are the franz-go and Kafka errors that indicate the region
// itself is unhealthy: records that could not be delivered in time, buffers that
// filled up because nothing is draining, and broker availability problems.
var regionFailureErrors = []error{
	kgo.ErrRecordTimeout,
	kgo.ErrRecordRetries,
	kgo.ErrMaxBuffered,
	context.DeadlineExceeded,
	io.EOF,
	io.ErrUnexpectedEOF,
	kerr.RequestTimedOut,
	kerr.LeaderNotAvailable,
	kerr.NotLeaderForPartition,
	kerr.BrokerNotAvailable,
	kerr.ReplicaNotAvailable,
	kerr.NetworkException,
	kerr.NotEnoughReplicas,
	kerr.NotEnoughReplicasAfterAppend,
}

// isRegionFailure reports whether err indicates a problem with the Kafka region.
// This is an allowlist: wrpkafka classifies every Kafka client error as "unknown",
// so the franz-go error is matched directly. Anything else, such as encoding or
// routing errors, per-message broker rejections, caller cancellation and client
// shutdown, says nothing about the region's health and is ignored.
func isRegionFailure(err error) bool {
	for _, regionErr := range regionFailureErrors {
		if errors.Is(err, regionErr) {
			return true
		}
	}

	// Connection failures (refused, reset, DNS, dial timeouts).
	var netErr net.Error
	return errors.As(err, &netErr)
}

// failoverGroup routes messages to the highest priority region whose circuit
// breaker is not open. The first region added is the preferred one.
// With no breakers configured it simply forwards to its only region.
type failoverGroup struct {
	regions       []*region
	active        atomic.Int32
	logEmitter    *observe.Subject[log.Event]
	metricEmitter *observe.Subject[metrics.Event]

	// refreshInterval is how often the background loop re-evaluates breaker state.
	refreshInterval time.Duration
	stopRefresh     context.CancelFunc
	refreshDone     chan struct{}
}

// defaultRefreshInterval is how often breaker state and gauges are refreshed.
const defaultRefreshInterval = time.Second

func newFailoverGroup(logEmitter *observe.Subject[log.Event], metricEmitter *observe.Subject[metrics.Event]) *failoverGroup {
	return &failoverGroup{
		logEmitter:      logEmitter,
		metricEmitter:   metricEmitter,
		refreshInterval: defaultRefreshInterval,
	}
}

// add registers a region in priority order. A nil cfg disables the breaker.
func (g *failoverGroup) add(name string, p producer, cfg *FailoverConfig) {
	r := &region{name: name, producer: p}

	if cfg != nil {
		r.breaker = gobreaker.NewCircuitBreaker[struct{}](gobreaker.Settings{
			Name:        name,
			MaxRequests: cfg.HalfOpenSuccesses,
			Interval:    cfg.Interval,
			Timeout:     cfg.OpenTimeout,
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				return counts.ConsecutiveFailures >= cfg.ConsecutiveFailures
			},
			OnStateChange: func(name string, from, to gobreaker.State) {
				g.onStateChange(name, from, to)
			},
		})
	}

	g.regions = append(g.regions, r)
}

// start registers listeners and starts every region. The primary region's client
// and every standby client are started up front so failover is immediate.
func (g *failoverGroup) start(listener func(*wrpkafka.PublishEvent)) error {
	for i, r := range g.regions {
		r.detach = append(r.detach, r.producer.AddPublishEventListener(listener))
		if r.breaker != nil {
			r.detach = append(r.detach, r.producer.AddPublishEventListener(r.record))
		}

		if err := r.producer.Start(); err != nil {
			g.detachListeners(g.regions[:i+1])
			g.stopRegions(context.Background(), g.regions[:i])
			return fmt.Errorf("failed to start region %s: %w", r.name, err)
		}
	}

	g.active.Store(0)
	for i, r := range g.regions {
		g.emitGauge(metrics.KafkaActiveRegion, r.name, boolToFloat(i == 0))
		if r.breaker != nil {
			g.emitGauge(metrics.KafkaRegionCircuitState, r.name, stateValue(gobreaker.StateClosed))
		}
	}

	if g.hasBreakers() {
		refreshCtx, cancel := context.WithCancel(context.Background())
		g.stopRefresh = cancel
		g.refreshDone = make(chan struct{})
		go g.refreshLoop(refreshCtx, g.refreshDone)
	}

	return nil
}

func (g *failoverGroup) hasBreakers() bool {
	for _, r := range g.regions {
		if r.breaker != nil {
			return true
		}
	}
	return false
}

// refreshLoop keeps breaker state, the active region and their metrics current
// when no messages are flowing. gobreaker only moves an expired open breaker to
// half-open when it is asked for its state, and failover is otherwise only
// noticed when a message is produced.
func (g *failoverGroup) refreshLoop(ctx context.Context, done chan<- struct{}) {
	defer close(done)

	ticker := time.NewTicker(g.refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			g.refresh()
		}
	}
}

// refresh re-selects the active region, which records any failover or failback,
// and re-publishes the gauges so they self-correct if an update was missed.
func (g *failoverGroup) refresh() {
	g.selectRegion()

	active := int(g.active.Load())
	for i, r := range g.regions {
		g.emitGauge(metrics.KafkaActiveRegion, r.name, boolToFloat(i == active))
		if r.breaker != nil {
			g.emitGauge(metrics.KafkaRegionCircuitState, r.name, stateValue(r.breaker.State()))
		}
	}
}

func (g *failoverGroup) stop(ctx context.Context) {
	if g.stopRefresh != nil {
		g.stopRefresh()
		<-g.refreshDone
		g.stopRefresh = nil
	}

	g.detachListeners(g.regions)
	g.stopRegions(ctx, g.regions)
}

func (g *failoverGroup) detachListeners(regions []*region) {
	for _, r := range regions {
		for _, detach := range r.detach {
			detach()
		}
		r.detach = nil
	}
}

func (g *failoverGroup) stopRegions(ctx context.Context, regions []*region) {
	for _, r := range regions {
		r.producer.Stop(ctx)
	}
}

// produce sends msg to the highest priority region that is not open.
func (g *failoverGroup) produce(ctx context.Context, msg *wrp.Message) (wrpkafka.Outcome, error) {
	r := g.selectRegion()
	if r == nil {
		g.logEmitter.Notify(log.NewEvent(log.LevelError, "all kafka regions are unavailable", nil))
		g.metricEmitter.Notify(metrics.Event{
			Name:  metrics.PublisherErrorsCounter,
			Value: 1,
			Labels: []string{
				metrics.ErrorTypeLabel, "no_region_available",
				metrics.TopicLabel, metrics.GetUnknownTagIfEmpty(""),
				metrics.TopicShardStrategyLabel, metrics.GetUnknownTagIfEmpty(""),
			},
		})
		return wrpkafka.Failed, ErrNoRegionAvailable
	}

	return r.producer.Produce(ctx, msg)
}

// bufferedRecords sums buffer usage across all regions.
func (g *failoverGroup) bufferedRecords() (currentRecords, maxRecords int, currentBytes, maxBytes int64) {
	for _, r := range g.regions {
		cr, mr, cb, mb := r.producer.BufferedRecords()
		currentRecords += cr
		maxRecords += mr
		currentBytes += cb
		maxBytes += mb
	}
	return currentRecords, maxRecords, currentBytes, maxBytes
}

// selectRegion returns the first region whose breaker is not open, recording a
// failover or failback when that differs from the previous selection.
func (g *failoverGroup) selectRegion() *region {
	for i, r := range g.regions {
		if r.isOpen() {
			continue
		}
		if prev := int(g.active.Swap(int32(i))); prev != i {
			g.onActiveChange(prev, i)
		}
		return r
	}
	return nil
}

func (g *failoverGroup) onActiveChange(from, to int) {
	fromName, toName := g.regions[from].name, g.regions[to].name

	level, msg := log.LevelWarn, "kafka region failover"
	if to == 0 {
		level, msg = log.LevelInfo, "kafka region failback"
	}
	g.logEmitter.NotifySync(log.NewEvent(level, msg, map[string]any{
		"from_region": fromName,
		"to_region":   toName,
	}))

	g.metricEmitter.NotifySync(metrics.Event{
		Name:   metrics.KafkaRegionFailovers,
		Value:  1,
		Labels: []string{metrics.FromRegionLabel, fromName, metrics.ToRegionLabel, toName},
	})
	g.emitGauge(metrics.KafkaActiveRegion, fromName, 0)
	g.emitGauge(metrics.KafkaActiveRegion, toName, 1)
}

// onStateChange is called by gobreaker with its lock held, so it must not call
// back into the breaker. Events are delivered synchronously, because gauge values
// from concurrently dispatched events could otherwise be applied out of order.
func (g *failoverGroup) onStateChange(name string, from, to gobreaker.State) {
	level := log.LevelInfo
	if to == gobreaker.StateOpen {
		level = log.LevelWarn
	}
	g.logEmitter.NotifySync(log.NewEvent(level, "kafka region circuit breaker state changed", map[string]any{
		"region":     name,
		"from_state": from.String(),
		"to_state":   to.String(),
	}))

	g.metricEmitter.NotifySync(metrics.Event{
		Name:  metrics.KafkaRegionCircuitTransitions,
		Value: 1,
		Labels: []string{
			metrics.RegionLabel, name,
			metrics.FromStateLabel, from.String(),
			metrics.ToStateLabel, to.String(),
		},
	})
	g.emitGauge(metrics.KafkaRegionCircuitState, name, stateValue(to))
}

func (g *failoverGroup) emitGauge(name, regionName string, value float64) {
	g.metricEmitter.NotifySync(metrics.Event{
		Name:   name,
		Value:  value,
		Labels: []string{metrics.RegionLabel, regionName},
	})
}

// stateValue maps a breaker state to the kafka_region_circuit_state gauge value.
func stateValue(s gobreaker.State) float64 {
	switch s {
	case gobreaker.StateHalfOpen:
		return 1
	case gobreaker.StateOpen:
		return 2
	default:
		return 0
	}
}

func boolToFloat(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
