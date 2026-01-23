package store

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

func TestNewShardedMapStore(t *testing.T) {
	tests := []struct {
		name        string
		shards      uint
		expectError bool
	}{
		{"Valid 1 shard", 1, false},
		{"Valid 2 shards", 2, false},
		{"Valid 64 shards", 64, false},
		{"Invalid 0 shards", 0, true},
		{"Invalid 3 shards (not power of 2)", 3, true},
		{"Invalid 63 shards (not power of 2)", 63, true},
		{"Invalid 128 shards (too many)", 128, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := NewShardedMapStore(tt.shards)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error for %d shards, got nil", tt.shards)
				}
				if s != nil {
					t.Errorf("expected nil struct for error case, got %v", s)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for %d shards: %v", tt.shards, err)
				}
				if uint(len(s.shards)) != tt.shards {
					t.Errorf("expected %d shards created, got %d", tt.shards, len(s.shards))
				}
				if s.shardMask != uint32(tt.shards-1) {
					t.Errorf("mask mismatch")
				}
			}
		})
	}
}

func TestShardedMapStore_Distribution(t *testing.T) {
	shardsCount := uint(16)
	store, _ := NewShardedMapStore(shardsCount) //nolint:errcheck

	keysPopulated := make(map[int]int)

	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key-%d", i)
		store.Set(key, "val", SetOptions{
			TTL:     0,
			KeepTTL: false,
			NX:      false,
			XX:      false,
		})

		shardIdx := store.getShardIndex(key)

		if _, ok := store.shards[shardIdx].Get(key); !ok {
			t.Errorf("Key %s hashed to shard %d but not found there", key, shardIdx)
		}
		keysPopulated[int(shardIdx)]++
	}

	if len(keysPopulated) < int(shardsCount) {
		t.Logf("Warning: Not all shards were used with 100 keys. Used: %d/%d.", len(keysPopulated), shardsCount)
	}
}

func TestShardedMapStore_Concurrent(t *testing.T) {
	store, _ := NewShardedMapStore(16) //nolint:errcheck
	var wg sync.WaitGroup

	workers := 100
	ops := 100000

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func(id int) {
			defer wg.Done()
			r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(id)))

			for j := 0; j < ops; j++ {
				key := fmt.Sprintf("key-%d", r.Intn(100))
				action := r.Intn(3)

				switch action {
				case 0:
					store.Set(key, fmt.Sprintf("val-%d", j), SetOptions{
						TTL:     0,
						KeepTTL: false,
						NX:      false,
						XX:      false,
					})
				case 1:
					store.Get(key)
				case 2:
					store.Delete(key)
				}
			}
		}(i)
	}

	wg.Wait()
}

func FuzzSharedMapStore(f *testing.F) {
	f.Add("key1", "val1")
	f.Add("special", "!@#$%^&*()")

	s, _ := NewShardedMapStore(8) //nolint:errcheck

	f.Fuzz(func(t *testing.T, key string, val string) {
		s.Set(key, val, SetOptions{
			TTL:     0,
			KeepTTL: false,
			NX:      false,
			XX:      false,
		})

		v, ok := s.Get(key)
		if !ok || v != val {
			t.Errorf("Get failed after Set: key=%q, val=%q", key, val)
		}
	})
}
