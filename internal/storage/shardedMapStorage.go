package storage

import (
	"errors"
	"hash/fnv"
	"io"
	"math/bits"
	"sync"
	"time"
)

// ShardedMapStorage is a thread-safe key-value storage,
// divided into segments (shards) to reduce contention for locking
type ShardedMapStorage struct {
	shards    []*MapStorage
	shardMask uint32
}

// NewShardedMapStorage creates a new instance of ShardedMapStorage.
// The requestedShards parameter must be a power of two for efficient allocation.
// The maximum allowed number of shards is 64.
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

// getShardIndex returns index of shard by key
func (s *ShardedMapStorage) getShardIndex(key string) uint32 {
	hash := fnv.New32a()
	hash.Write([]byte(key)) //nolint:errcheck

	return hash.Sum32() & s.shardMask
}

// Get returns the value and true if the key is found. Otherwise, "", false.
func (s *ShardedMapStorage) Get(key string) (string, bool, error) {
	return s.shards[s.getShardIndex(key)].Get(key)
}

// Set writes the value based on the options. Returns true if recording has been performed.
func (s *ShardedMapStorage) Set(key, value string, options SetOptions) bool {
	return s.shards[s.getShardIndex(key)].Set(key, value, options)
}

// Delete deletes the key. Returns true if the key existed and was deleted.
func (s *ShardedMapStorage) Delete(key string) bool {
	return s.shards[s.getShardIndex(key)].Delete(key)
}

// Expiry returns the remaining lifetime and status as ExpiryStatus
func (s *ShardedMapStorage) Expiry(key string) (time.Duration, ExpiryStatus) {
	return s.shards[s.getShardIndex(key)].Expiry(key)
}

// Persist removes the expiration date of the key, making it eternal.
// Returns 1 if successful, 0 if the key was not found or had no TTL
func (s *ShardedMapStorage) Persist(key string) int64 {
	return s.shards[s.getShardIndex(key)].Persist(key)
}

// DeleteExpired randomly selects a limit of keys from each shard and delete if his TTL has expired
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

// Snapshot iterates over all shards sequentially to minimize locking time
func (s *ShardedMapStorage) Snapshot(w io.Writer) error {
	for _, shard := range s.shards {
		if err := shard.Snapshot(w); err != nil {
			return err
		}
	}
	return nil
}

// Restore reads the stream and fills the maps
func (s *ShardedMapStorage) Restore(r io.Reader) error {
	tempLoader := NewMapStorage()
	if err := tempLoader.Restore(r); err != nil {
		return err
	}

	tempLoader.mu.RLock()
	defer tempLoader.mu.RUnlock()

	for key, val := range tempLoader.data {
		expire := tempLoader.expires[key]

		targetShard := s.shards[s.getShardIndex(key)]
		targetShard.mu.Lock()
		targetShard.data[key] = val
		if expire > 0 {
			targetShard.expires[key] = expire
		}
		targetShard.mu.Unlock()
	}

	return nil
}

// HSet sets the specified fields to their respective values in the hash stored at key
func (s *ShardedMapStorage) HSet(key string, fields map[string]string) int64 {
	return s.shards[s.getShardIndex(key)].HSet(key, fields)
}

// HGet returns the value associated with field in the hash stored at key
func (s *ShardedMapStorage) HGet(key, field string) (string, bool) {
	return s.shards[s.getShardIndex(key)].HGet(key, field)
}

// HGetAll returns all fields and values of the hash stored at key
func (s *ShardedMapStorage) HGetAll(key string) map[string]string {
	return s.shards[s.getShardIndex(key)].HGetAll(key)
}

// HDel calculate index shard and delegates all the logic of the work to the MapStorage
func (s *ShardedMapStorage) HDel(key string, fields []string) int64 {
	return s.shards[s.getShardIndex(key)].HDel(key, fields)
}

// HExists returns if field is an existing field in the hash stored at key
func (s *ShardedMapStorage) HExists(key, field string) int64 {
	return s.shards[s.getShardIndex(key)].HExists(key, field)
}

// HLen returns the number of fields contained in the hash stored at key
func (s *ShardedMapStorage) HLen(key string) int64 {
	return s.shards[s.getShardIndex(key)].HLen(key)
}

// HKeys returns all field names in the hash stored at key
func (s *ShardedMapStorage) HKeys(key string) []string {
	return s.shards[s.getShardIndex(key)].HKeys(key)
}

// HVals returns all values in the hash stored at key
func (s *ShardedMapStorage) HVals(key string) []string {
	return s.shards[s.getShardIndex(key)].HVals(key)
}
