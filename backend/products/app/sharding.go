package app

import (
	"github.com/cespare/xxhash/v2"
)

func GetShardID(shards []ShardName, key string) int {
	maxScore := uint64(0)
	shardID := -1
	for i := range shards {
		score := xxhash.Sum64String(key + ":" + string(shards[i]))
		if score > maxScore {
			maxScore = score
			shardID = i
		}
	}
	return shardID
}
