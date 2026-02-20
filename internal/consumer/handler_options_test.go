// SPDX-FileCopyrightText: 2026 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package consumer

import (
	"testing"

	"xmidt-org/splitter/internal/log"
	"xmidt-org/splitter/internal/metrics"
	"xmidt-org/splitter/internal/observe"

	"github.com/stretchr/testify/assert"
)

// Use MockPublisher from mocks_test.go for publisher tests

func TestWithHandlerProducer(t *testing.T) {
	p := &MockPublisher{}
	h := &WRPMessageHandler{}
	err := WithHandlerProducer(p).apply(h)
	assert.NoError(t, err)
	assert.Equal(t, p, h.producer)

	err = WithHandlerProducer(nil).apply(h)
	assert.Error(t, err)
}

func TestWithHandlerLogEmitter(t *testing.T) {
	emitter := observe.NewSubject[log.Event]()
	h := &WRPMessageHandler{}
	err := WithHandlerLogEmitter(emitter).apply(h)
	assert.NoError(t, err)
	assert.Equal(t, emitter, h.logEmitter)

	err = WithHandlerLogEmitter(nil).apply(h)
	assert.Error(t, err)
}

func TestWithHandlerMetricsEmitter(t *testing.T) {
	emitter := observe.NewSubject[metrics.Event]()
	h := &WRPMessageHandler{}
	err := WithHandlerMetricsEmitter(emitter).apply(h)
	assert.NoError(t, err)
	assert.Equal(t, emitter, h.metricEmitter)

	err = WithHandlerMetricsEmitter(nil).apply(h)
	assert.Error(t, err)
}

func TestWithBuckets(t *testing.T) {
	b := &MockBuckets{}
	h := &WRPMessageHandler{}
	err := WithBuckets(b).apply(h)
	assert.NoError(t, err)
	assert.Equal(t, b, h.buckets)
}

func TestValidateHandler(t *testing.T) {
	h := &WRPMessageHandler{}
	err := validateHandler(h)
	assert.Error(t, err)

	h.producer = &MockPublisher{}
	err = validateHandler(h)
	assert.NoError(t, err)
}
