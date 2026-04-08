package storage

import "github.com/cespare/xxhash/v2"

type ShardName = string

type Shards[T any] map[ShardName]T

func (shards Shards[T]) Get(key string) (ShardName, T) {
	if len(shards) == 0 {
		panic("empty shards")
	}
	maxScore := uint64(0)
	var shardName ShardName
	for name := range shards {
		score := xxhash.Sum64String(key + ":" + string(name))
		if score > maxScore {
			maxScore = score
			shardName = name
		}
	}
	return shardName, shards[shardName]
}
