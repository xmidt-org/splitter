// SPDX-FileCopyrightText: 2026 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package bucket

import (
	"fmt"
	"sort"

	"github.com/xmidt-org/wrp-go/v3"
)

// TODO - make the hash algorithm configurable, but not yet a requirement

// this file determines whether or not a wrp.Message should be published to a target bucket.
// A splitter will only be configured to write to one bucket.  Messages destined for
// other buckets (regions) will be dropped.

type KeyType int

const (
	DeviceId KeyType = iota
)

const (
	DeviceIdKeyName = "device_id"
)

type Buckets struct {
	targetBucketIndex int32
	partitioner       Partitioner
	partitionKeyType  KeyType
	numBuckets        int32
}

func NewBuckets(targetBucket string, buckets []string, keyType string) (Buckets, error) {
	// sort buckets slice in alphabetical order
	sort.Strings(buckets)

	// create the partitioner
	partition := NewPartitioner()
	// set the index for the target bucket
	targetBucketIndex, err := getTargetIndex(targetBucket, buckets)
	if err != nil {
		return Buckets{}, err
	}

	// set the bucket key type
	partitionKeyType, err := getPartitionKeyType(keyType)
	if err != nil {
		return Buckets{}, err
	}

	return Buckets{
		partitioner:       partition,
		targetBucketIndex: int32(targetBucketIndex),
		numBuckets:        int32(len(buckets)),
		partitionKeyType:  partitionKeyType}, nil
}

func getTargetIndex(targetBucket string, buckets []string) (int, error) {
	for i, bucket := range buckets {
		if bucket == targetBucket {
			return i, nil
		}
	}
	return -1, fmt.Errorf("target bucket %s not found", targetBucket)
}

// only process messages that hash to the target bucket
func (r *Buckets) ShouldPublish(msg *wrp.Message) bool {
	partitionKey, err := r.getPartitionKey(msg)
	if err != nil {
		return false
	}
	bucket, err := r.partitioner.Partition(partitionKey, r.numBuckets)
	if err != nil {
		return false
	}
	return bucket == r.targetBucketIndex
}

func (r *Buckets) getPartitionKey(msg *wrp.Message) (string, error) {
	switch r.partitionKeyType {
	case DeviceId:
		return parseDeviceId(msg)
	default:
		return "", fmt.Errorf("invalid partition key type")
	}
}

func parseDeviceId(msg *wrp.Message) (string, error) {
	deviceID, err := wrp.ParseDeviceID(msg.Source)
	if err != nil {
		return "", fmt.Errorf("invalid device ID in WRP Source `%s`: %w", msg.Source, err)
	}
	return string(deviceID), nil
}

func getPartitionKeyType(partitionKey string) (KeyType, error) {
	if partitionKey == DeviceIdKeyName {
		return DeviceId, nil
	}
	return -1, fmt.Errorf("invalid bucket hash key %s", partitionKey)
}
