package sharding

import "github.com/cespare/xxhash/v2"

type ShardName = string

func GetShard[V any](shards map[ShardName]V, key string) (ShardName, V) {
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
