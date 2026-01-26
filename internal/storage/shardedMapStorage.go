package storage

import (
	"errors"
	"hash/fnv"
	"math/bits"
	"sync"
	"time"
)

type ShardedMapStorage struct {
	shards    []*MapStorage
	shardMask uint32
}

func NewShardedMapStorage(requestedShards uint) (*ShardedMapStorage, error) {
	if bits.OnesCount(requestedShards) != 1 {
		return nil, errors.New("requested shards must be a power of 2")
	}

	if requestedShards > 64 {
		return nil, errors.New("requested shards must be less or equal than 64")
	}

	s := &ShardedMapStorage{
		shards:    make([]*MapStorage, requestedShards),
		shardMask: uint32(requestedShards - 1),
	}

	var i uint
	for i = 0; i < requestedShards; i++ {
		s.shards[i] = NewMapStorage()
	}

	return s, nil
}

func (s *ShardedMapStorage) getShardIndex(key string) uint32 {
	hash := fnv.New32a()
	hash.Write([]byte(key)) //nolint:errcheck

	return hash.Sum32() & s.shardMask
}

func (s *ShardedMapStorage) Get(key string) (string, bool) {
	return s.shards[s.getShardIndex(key)].Get(key)
}

func (s *ShardedMapStorage) Set(key, value string, options SetOptions) bool {
	return s.shards[s.getShardIndex(key)].Set(key, value, options)
}

func (s *ShardedMapStorage) Delete(key string) bool {
	return s.shards[s.getShardIndex(key)].Delete(key)
}

func (s *ShardedMapStorage) Expiry(key string) (time.Duration, int) {
	return s.shards[s.getShardIndex(key)].Expiry(key)
}

func (s *ShardedMapStorage) Persist(key string) int64 {
	return s.shards[s.getShardIndex(key)].Persist(key)
}

func (s *ShardedMapStorage) DeleteExpired(limit int) float64 {
	var wg sync.WaitGroup
	var totalRatio float64
	var mu sync.Mutex // protects totalRatio

	shardCount := len(s.shards)
	wg.Add(shardCount)

	for _, shard := range s.shards {
		go func(m *MapStorage) {
			ratio := m.DeleteExpired(limit)

			mu.Lock()
			totalRatio += ratio
			mu.Unlock()

			wg.Done()
		}(shard)
	}

	wg.Wait()

	return totalRatio / float64(shardCount)
}
