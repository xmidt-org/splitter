package bucket

import (
	"errors"

	"github.com/aviddiviner/go-murmur"
)

// this is just abstract buckets to determine which cluster / region to go to,
// not really anything to do with actual Kafka partitioning
type Partitioner struct{}

// NewPartitionerConstructor returns a PartitionerConstructor that selects Murmur2 for topic with
// prefix with any of the murmur2topics, otherwise uses default Sarama HashParitition for others.
func NewPartitioner() Partitioner {
	return Partitioner{}
}

func (p *Partitioner) Partition(key string, numPartitions int32) (int32, error) {
	const seed uint32 = 0x9747b28c // do we need this seed since it is kafka specific?
	// these records are thrown out later, anyway
	if key == "" {
		return 0, errors.New("no key specified for bucket")
	}
	bytes := []byte(key)

	hash := murmur.MurmurHash2(bytes, seed)
	part := toPositive(hash) % numPartitions
	return part, nil
}

// toPositive returns positive value as in Java Kafka client:
// https://github.com/apache/kafka/blob/1.0.0/clients/src/main/java/org/apache/kafka/common/utils/Utils.java#L728
func toPositive(n uint32) int32 {
	return int32(n) & 0x7fffffff
}
