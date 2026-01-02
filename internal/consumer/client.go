// SPDX-FileCopyrightText: 2026 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package consumer

import (
	"context"

	"github.com/twmb/franz-go/pkg/kgo"
)

// Client defines the interface for a Kafka client.
// This interface allows for mocking in tests while using the real kgo.Client in production.
type Client interface {
	// Ping pings the broker to verify connectivity.
	Ping(ctx context.Context) error

	// PollFetches polls for new records from Kafka.
	PollFetches(ctx context.Context) kgo.Fetches

	// CommitUncommittedOffsets commits any uncommitted offsets.
	CommitUncommittedOffsets(ctx context.Context) error

	// Close closes the client and cleans up resources.
	Close()
}

// Ensure kgo.Client implements our Client interface at compile time.
var _ Client = (*kgo.Client)(nil)
