// SPDX-FileCopyrightText: 2026 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package bucket

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type PartitionerSuite struct {
	suite.Suite
	partitioner Partitioner
}

func (s *PartitionerSuite) SetupTest() {
	s.partitioner = NewPartitioner()
}

func (s *PartitionerSuite) TestPartition_Deterministic() {
	thresholds := []float32{0.3, 0.6, 1.0}
	key := "device123"
	bucket1, err1 := s.partitioner.Partition(key, thresholds)
	bucket2, err2 := s.partitioner.Partition(key, thresholds)
	assert.NoError(s.T(), err1)
	assert.NoError(s.T(), err2)
	assert.Equal(s.T(), bucket1, bucket2, "Partitioning should be deterministic for the same key and thresholds")
}

func (s *PartitionerSuite) TestPartition_ThresholdAssignment() {
	thresholds := []float32{0.2, 0.5, 0.8, 1.0}
	// Try a range of keys and ensure all buckets are possible
	bucketSeen := make(map[int]bool)
	for i := 0; i < 1000; i++ {
		key := string(rune(i))
		bucket, err := s.partitioner.Partition(key, thresholds)
		assert.NoError(s.T(), err)
		assert.GreaterOrEqual(s.T(), bucket, 0)
		assert.Less(s.T(), bucket, len(thresholds))
		bucketSeen[bucket] = true
	}
	// All buckets should be seen
	for i := 0; i < len(thresholds); i++ {
		assert.True(s.T(), bucketSeen[i], "Bucket %d should be assigned", i)
	}
}

func (s *PartitionerSuite) TestPartition_EmptyKey() {
	thresholds := []float32{0.5, 1.0}
	bucket, err := s.partitioner.Partition("", thresholds)
	assert.Error(s.T(), err)
	assert.Equal(s.T(), 0, bucket)
}

func (s *PartitionerSuite) TestPartition_EmptyThresholds() {
	bucket, err := s.partitioner.Partition("device123", []float32{})
	assert.Error(s.T(), err)
	assert.Equal(s.T(), 0, bucket)
}

func (s *PartitionerSuite) TestPartition_ThresholdsAscending() {
	// Should work with strictly ascending thresholds in (0,1]
	thresholds := []float32{0.1, 0.5, 0.9, 1.0}
	bucket, err := s.partitioner.Partition("device123", thresholds)
	assert.NoError(s.T(), err)
	assert.GreaterOrEqual(s.T(), bucket, 0)
	assert.Less(s.T(), bucket, len(thresholds))
}

func TestPartitionerSuite(t *testing.T) {
	suite.Run(t, new(PartitionerSuite))
}
