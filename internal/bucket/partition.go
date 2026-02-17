// SPDX-FileCopyrightText: 2026 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package bucket

import (
	"errors"

	"github.com/aviddiviner/go-murmur"
)

// specifies a partition based on key and number of partitions / buckets.
// TODO - the seed is kafka specific, do we need this since this is not
// directly related to kafka partitioning?

type Partitioner struct{}

func NewPartitioner() Partitioner {
	return Partitioner{}
}

func (p *Partitioner) Partition(key string, thresholds []float32) (int, error) {
	const seed uint32 = 0x9747b28c
	if key == "" {
		return 0, errors.New("no bucket key specified")
	}
	if len(thresholds) == 0 {
		return 0, errors.New("no thresholds provided")
	}
	bytes := []byte(key)
	hash := murmur.MurmurHash2(bytes, seed)
	// Convert hash to float in [0,1)
	hashFloat := float32(toPositive(hash)) / float32(0x7fffffff)
	for i, threshold := range thresholds {
		if hashFloat < threshold {
			return i, nil
		}
	}
	// If not less than any threshold, assign to last bucket
	return len(thresholds) - 1, nil
}

// TODO - do we need to be compatible with kafka here? Outstanding question to xdp
// toPositive returns positive value as in Java Kafka client:
// https://github.com/apache/kafka/blob/1.0.0/clients/src/main/java/org/apache/kafka/common/utils/Utils.java#L728
func toPositive(n uint32) int32 {
	// #nosec G115 -- conversion is safe and intentional for Kafka compatibility
	return int32(n) & 0x7fffffff
}
