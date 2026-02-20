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
// A splitter will only be configured to write to one "bucket".  Messages destined for
// other buckets (in this case, regions) will be dropped.

type KeyType int

const (
	DeviceId KeyType = iota
)

const (
	DeviceIdKeyName = "device_id"
)

type Config struct {
	TargetBucket    string
	PossibleBuckets []Bucket

	// partition key type determines which field of the message is used for hashing to a bucket
	PartitionKeyType string
}

type Bucket struct {
	Name      string
	Threshold float32
}

type Buckets struct {
	targetBucketIndex int
	partitioner       Partitioner
	partitionKeyType  KeyType
	buckets           []Bucket
	thresholds        []float32
}

func NewBuckets(config Config) (Buckets, error) {
	if (len(config.PossibleBuckets) == 0) {
		// if there are no buckets, then all messages are in the target bucket
		return Buckets{}, nil
	}
	
	// sort buckets slice in order of threshold, ascending
	sort.Slice(config.PossibleBuckets, func(i, j int) bool {
		return config.PossibleBuckets[i].Threshold < config.PossibleBuckets[j].Threshold
	})

	// for convenience, extract ordered thresholds for the partitioner
	thresholds := make([]float32, len(config.PossibleBuckets))
	for i, bucket := range config.PossibleBuckets {
		thresholds[i] = bucket.Threshold
	}

	// create the partitioner
	partition := NewPartitioner()

	// set the index for the target bucket
	targetBucketIndex, err := getTargetIndex(config.TargetBucket, config.PossibleBuckets)
	if err != nil {
		return Buckets{}, err
	}

	// set the bucket key type
	partitionKeyType, err := getPartitionKeyType(config.PartitionKeyType)
	if err != nil {
		return Buckets{}, err
	}

	return Buckets{
		partitioner:       partition,
		targetBucketIndex: targetBucketIndex,
		buckets:           config.PossibleBuckets,
		thresholds:        thresholds,
		partitionKeyType:  partitionKeyType}, nil
}

func getTargetIndex(targetBucket string, buckets []Bucket) (int, error) {
	for i, bucket := range buckets {
		if bucket.Name == targetBucket {
			return i, nil
		}
	}
	return -1, fmt.Errorf("target bucket %s not found", targetBucket)
}

// determine if message hashes to the target bucket
func (r *Buckets) IsInTargetBucket(msg *wrp.Message) bool {
	// if there are no buckets, there is no hashing
	if (len(r.buckets) == 0) {
		return true
	}

	partitionKey, err := r.getPartitionKey(msg)
	if err != nil {
		return false
	}
	bucket, err := r.partitioner.Partition(partitionKey, r.thresholds)
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
