package sharding

import (
	"fmt"
	"maps"
	"slices"

	"github.com/cespare/xxhash/v2"
)

type ShardName string

type Resolver func(shardNames []ShardName, key string) ShardName

func RendezvousResolver(shardNames []ShardName, key string) ShardName {
	maxScore := uint64(0)
	var res ShardName
	for _, name := range shardNames {
		score := xxhash.Sum64String(key + ":" + string(name))
		if score > maxScore {
			maxScore = score
			res = name
		}
	}
	return res
}

type Pool[T any] struct {
	shards   map[ShardName]T
	resolver Resolver
}

func NewPool[T any](shards map[ShardName]T, resolver Resolver) (*Pool[T], error) {
	if len(shards) == 0 {
		return nil, fmt.Errorf("empty shards")
	}
	return &Pool[T]{shards: shards, resolver: resolver}, nil
}

func (s *Pool[T]) Get(key string) T {
	shardName := s.resolver(slices.Collect(maps.Keys(s.shards)), key)
	return s.shards[shardName]
}

func (s *Pool[T]) GetByName(name ShardName) T {
	return s.shards[name]
}

func (s *Pool[T]) GetName(key string) ShardName {
	shardName := s.resolver(slices.Collect(maps.Keys(s.shards)), key)
	return shardName
}

func (s *Pool[T]) All() []T {
	all := make([]T, 0, len(s.shards))
	for _, v := range s.shards {
		all = append(all, v)
	}
	return all
}
