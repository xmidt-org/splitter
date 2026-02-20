// SPDX-FileCopyrightText: 2026 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package consumer

import (
	"errors"
	"xmidt-org/splitter/internal/bucket"
	"xmidt-org/splitter/internal/log"
	"xmidt-org/splitter/internal/metrics"
	"xmidt-org/splitter/internal/observe"
	"xmidt-org/splitter/internal/publisher"
)

type Handlerption interface {
	apply(*WRPMessageHandler) error
}

type handlerOptionFunc func(*WRPMessageHandler) error

var _ HandlerOption = handlerOptionFunc(nil)

func (f handlerOptionFunc) apply(h *WRPMessageHandler) error {
	return f(h)
}

// HandlerOption configures a WRPMessageHandlerConfig.
type HandlerOption interface {
	apply(*WRPMessageHandler) error
}

func validateHandler(h *WRPMessageHandler) error {
	if h.producer == nil {
		return errors.New("producer is required")
	}
	return nil
}

func WithHandlerProducer(p publisher.Publisher) HandlerOption {
	return handlerOptionFunc(func(h *WRPMessageHandler) error {
		if p == nil {
			return errors.New("producer cannot be nil")
		}
		h.producer = p
		return nil
	})
}

// WithHandlerLogEmitter sets the LogEmitter for WRPMessageHandlerConfig.
func WithHandlerLogEmitter(l *observe.Subject[log.Event]) HandlerOption {
	return handlerOptionFunc(func(h *WRPMessageHandler) error {
		if l == nil {
			return errors.New("log emitter cannot be nil")
		}
		h.logEmitter = l
		return nil
	})
}

// WithHandlerMetricsEmitter sets the MetricsEmitter for WRPMessageHandlerConfig.
func WithHandlerMetricsEmitter(m *observe.Subject[metrics.Event]) HandlerOption {
	return handlerOptionFunc(func(h *WRPMessageHandler) error {
		if m == nil {
			return errors.New("metrics emitter cannot be nil")
		}
		h.metricEmitter = m
		return nil
	})
}

func WithBuckets(b bucket.Bucket) HandlerOption {
	return handlerOptionFunc(func(h *WRPMessageHandler) error {
		h.buckets = b
		return nil
	})
}
