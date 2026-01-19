package store

import (
	"errors"
	"hash/fnv"
	"math/bits"
)

type ShardedMapStore struct {
	shards    []*MapStore
	shardMask uint32
}

func NewShardedMapStore(requestedShards uint) (*ShardedMapStore, error) {
	if bits.OnesCount(requestedShards) != 1 {
		return nil, errors.New("requested shards must be a power of 2")
	}

	if requestedShards > 64 {
		return nil, errors.New("requested shards must be less or equal than 64")
	}

	s := &ShardedMapStore{
		shards:    make([]*MapStore, requestedShards),
		shardMask: uint32(requestedShards - 1),
	}

	var i uint
	for i = 0; i < requestedShards; i++ {
		s.shards[i] = NewMapStore()
	}

	return s, nil
}

func (s *ShardedMapStore) getShardIndex(key string) uint32 {
	hash := fnv.New32a()
	hash.Write([]byte(key)) //nolint:errcheck

	return hash.Sum32() & s.shardMask
}

func (s *ShardedMapStore) Get(key string) (string, bool) {
	return s.shards[s.getShardIndex(key)].Get(key)
}

func (s *ShardedMapStore) Set(key, value string) {
	s.shards[s.getShardIndex(key)].Set(key, value)
}

func (s *ShardedMapStore) Delete(key string) {
	s.shards[s.getShardIndex(key)].Delete(key)
}
