// SPDX-FileCopyrightText: 2026 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package bucket

import (
	"fmt"
	"sort"

	"github.com/xmidt-org/wrp-go/v5"
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

type MissingPartitionKeyAction int

const (
	Drop MissingPartitionKeyAction = iota
	IncludeInBucket
)

const (
	DefaultMissingPartitionKeyAction = IncludeInBucket
)

const (
	DropMissingPartitionKeyActionName            = "drop"
	IncludeInBucketMissingPartitionKeyActionName = "include"
)

var (
	ErrNoPartitionKey = fmt.Errorf("no partition key found")
)

type Bucket interface {
	IsInTargetBucket(msg *wrp.Message) (bool, error)
}

type Config struct {
	TargetBucket    string
	PossibleBuckets []BucketSettings

	// partition key type determines which field of the message is used for hashing to a bucket
	PartitionKeyType string

	// what to do when the partition key is missing from the message - either drop the message or include it in the target bucket
	MissingPartitionKeyAction string
}

type BucketSettings struct {
	Name      string
	Threshold float32
}

type Buckets struct {
	targetBucketIndex         int
	partitioner               Partitioner
	missingPartitionKeyAction MissingPartitionKeyAction
	partitionKeyType          KeyType
	buckets                   []BucketSettings
	thresholds                []float32
}

func NewBuckets(config Config) (Bucket, error) {
	if len(config.PossibleBuckets) == 0 {
		// if there are no buckets, then all messages are in the target bucket
		return &Buckets{}, nil
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
		return &Buckets{}, err
	}

	// set the bucket key type
	partitionKeyType, err := getPartitionKeyType(config.PartitionKeyType)
	if err != nil {
		return &Buckets{}, err
	}

	// set the missing partition key action
	missingPartitionKeyAction := DefaultMissingPartitionKeyAction
	if config.MissingPartitionKeyAction != "" {
		missingPartitionKeyAction, err = getMissingPartitionKeyAction(config.MissingPartitionKeyAction)
		if err != nil {
			return &Buckets{}, err
		}
	}

	return &Buckets{
		partitioner:               partition,
		targetBucketIndex:         targetBucketIndex,
		buckets:                   config.PossibleBuckets,
		thresholds:                thresholds,
		partitionKeyType:          partitionKeyType,
		missingPartitionKeyAction: missingPartitionKeyAction}, nil
}

func getTargetIndex(targetBucket string, buckets []BucketSettings) (int, error) {
	for i, bucket := range buckets {
		if bucket.Name == targetBucket {
			return i, nil
		}
	}
	return -1, fmt.Errorf("target bucket %s not found", targetBucket)
}

// determine if message hashes to the target bucket
func (r *Buckets) IsInTargetBucket(msg *wrp.Message) (bool, error) {
	// if there are no buckets or only one bucket, then all messages are in the target bucket
	if len(r.buckets) <= 1 {
		return true, nil
	}

	partitionKey, err := r.getPartitionKey(msg)
	if err != nil {
		if r.missingPartitionKeyAction == Drop {
			return false, err
		}
		return true, err
	}

	bucket, err := r.partitioner.Partition(partitionKey, r.thresholds)
	if err != nil {
		return false, err
	}

	return bucket == r.targetBucketIndex, nil
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
	// Try Source first
	deviceID, err := wrp.ParseDeviceID(msg.Source)
	if err != nil {
		// If Source fails, try Destination using ParseLocator
		locator, err := wrp.ParseLocator(msg.Destination)
		if err != nil {
			return "", fmt.Errorf("invalid device ID in both WRP Source `%s` and WRP Destination `%s`: %w", msg.Source, msg.Destination, err)
		}

		// For event locators, check if there's a device ID in the ignored/path portion
		if locator.Scheme == "event" && locator.Ignored != "" {
			// Try to parse the ignored portion as it may contain a device ID
			// Format: event:device-status/mac:112233445566/online
			// The ignored portion would be: /mac:112233445566/online
			deviceID, err := wrp.ParseDeviceID(locator.Ignored[1:]) // Skip leading '/'
			if err == nil {
				return string(deviceID), nil
			}
		}

		// Otherwise use the locator's ID (authority)
		if locator.ID != "" {
			return string(locator.ID), nil
		}

		return "", fmt.Errorf("no device ID found in WRP Source `%s` or Destination `%s`: %w", msg.Source, msg.Destination, ErrNoPartitionKey)
	}
	return string(deviceID), nil
}

func getPartitionKeyType(partitionKey string) (KeyType, error) {
	if partitionKey == DeviceIdKeyName {
		return DeviceId, nil
	}
	return -1, fmt.Errorf("invalid bucket hash key %s", partitionKey)
}

func getMissingPartitionKeyAction(action string) (MissingPartitionKeyAction, error) {
	switch action {
	case DropMissingPartitionKeyActionName:
		return Drop, nil
	case IncludeInBucketMissingPartitionKeyActionName:
		return IncludeInBucket, nil
	default:
		return -1, fmt.Errorf("invalid missing partition key action %s", action)
	}
}
