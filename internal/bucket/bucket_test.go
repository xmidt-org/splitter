// SPDX-FileCopyrightText: 2026 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package bucket

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/wrp-go/v3"
)

type BucketsSuite struct {
	suite.Suite
	buckets    Buckets
	bucketDefs []Bucket
}

func (s *BucketsSuite) SetupTest() {
	s.bucketDefs = []Bucket{
		{"bucketA", 0.33},
		{"bucketB", 0.66},
		{"bucketC", 1.0},
	}
	var err error
	s.buckets, err = NewBuckets("bucketB", s.bucketDefs, DeviceIdKeyName)
	s.Require().NoError(err)
}

func (s *BucketsSuite) TestNewBuckets_TargetIndex() {
	assert.Equal(s.T(), 1, s.buckets.targetBucketIndex)
	assert.Equal(s.T(), 3, len(s.buckets.buckets))
	assert.Equal(s.T(), 3, len(s.buckets.thresholds))
}

func (s *BucketsSuite) TestNewBuckets_InvalidTarget() {
	_, err := NewBuckets("notfound", s.bucketDefs, DeviceIdKeyName)
	assert.Error(s.T(), err)
}

func (s *BucketsSuite) TestNewBuckets_InvalidKeyType() {
	_, err := NewBuckets("bucketA", s.bucketDefs, "unknown_key")
	assert.Error(s.T(), err)
}

func (s *BucketsSuite) TestShouldPublish_True() {
	msg := &wrp.Message{Source: "mac:112233445566"}
	partitionKey, _ := s.buckets.getPartitionKey(msg)
	partitioner := NewPartitioner()
	bucket, _ := partitioner.Partition(partitionKey, s.buckets.thresholds)
	s.buckets.targetBucketIndex = bucket
	assert.True(s.T(), s.buckets.ShouldPublish(msg))
}

func (s *BucketsSuite) TestShouldPublish_False() {
	msg := &wrp.Message{Source: "mac:112233445566"}
	partitionKey, _ := s.buckets.getPartitionKey(msg)
	partitioner := NewPartitioner()
	bucket, _ := partitioner.Partition(partitionKey, s.buckets.thresholds)
	s.buckets.targetBucketIndex = (bucket + 1) % len(s.buckets.thresholds)
	assert.False(s.T(), s.buckets.ShouldPublish(msg))
}

func (s *BucketsSuite) TestShouldPublish_InvalidKey() {
	msg := &wrp.Message{Source: "invalid"}
	assert.False(s.T(), s.buckets.ShouldPublish(msg))
}

func (s *BucketsSuite) TestGetPartitionKey_DeviceId() {
	msg := &wrp.Message{Source: "mac:112233445566"}
	key, err := s.buckets.getPartitionKey(msg)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), "mac:112233445566", key)
}

func (s *BucketsSuite) TestGetPartitionKey_InvalidType() {
	b := s.buckets
	b.partitionKeyType = -1
	msg := &wrp.Message{Source: "mac:112233445566"}
	_, err := b.getPartitionKey(msg)
	assert.Error(s.T(), err)
}

func (s *BucketsSuite) TestParseDeviceId_Invalid() {
	msg := &wrp.Message{Source: "not-a-device-id"}
	_, err := parseDeviceId(msg)
	assert.Error(s.T(), err)
}

func (s *BucketsSuite) TestGetPartitionKeyType_Valid() {
	kt, err := getPartitionKeyType(DeviceIdKeyName)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), DeviceId, kt)
}

func (s *BucketsSuite) TestGetPartitionKeyType_Invalid() {
	_, err := getPartitionKeyType("bad_key")
	assert.Error(s.T(), err)
}

func TestBucketsSuite(t *testing.T) {
	suite.Run(t, new(BucketsSuite))
}
