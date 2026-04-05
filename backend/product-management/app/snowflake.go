package app

import (
	"math/rand"
	"sync"
	"time"
)

type Snowflake struct {
	mu       sync.Mutex
	lastTS   int64
	sequence int64

	nodeID int64

	epoch int64
}

const (
	nodeBits     = 10
	sequenceBits = 12

	maxSequence = (1 << sequenceBits) - 1
)

func NewSnowflake() *Snowflake {
	return &Snowflake{
		nodeID: rand.Int63n(1 << 10), // 10 бит
		epoch:  1700000000000,        // кастомная эпоха (ms)
	}
}

func (s *Snowflake) NextID() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UnixMilli()

	if now == s.lastTS {
		s.sequence = (s.sequence + 1) & maxSequence
		if s.sequence == 0 {
			// ждём следующую миллисекунду
			for now <= s.lastTS {
				now = time.Now().UnixMilli()
			}
		}
	} else {
		s.sequence = 0
	}

	s.lastTS = now

	return ((now - s.epoch) << (nodeBits + sequenceBits)) |
		(s.nodeID << sequenceBits) |
		s.sequence
}
