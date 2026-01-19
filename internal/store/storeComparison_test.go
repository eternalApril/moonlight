package store

import (
	"fmt"
	"testing"
)

func getAllImplementations() map[string]Storage {
	sharedMap16, _ := NewShardedMapStore(16)
	sharedMap32, _ := NewShardedMapStore(32)
	sharedMap64, _ := NewShardedMapStore(64)

	return map[string]Storage{
		"MapStore":          NewMapStore(),
		"SharedMapStore_16": sharedMap16,
		"SharedMapStore_32": sharedMap32,
		"SharedMapStore_64": sharedMap64,
	}
}

func BenchmarkStorage(b *testing.B) {
	implementations := getAllImplementations()

	for name, s := range implementations {
		b.Run(fmt.Sprintf("%s/ReadOnly", name), func(b *testing.B) {
			s.Set("bench_key", "value")
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					s.Get("bench_key")
				}
			})
		})

		b.Run(fmt.Sprintf("%s/Mixed90-10", name), func(b *testing.B) {
			keyCount := 1000
			for i := 0; i < keyCount; i++ {
				s.Set(fmt.Sprintf("key%d", i), "val")
			}
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					key := fmt.Sprintf("key%d", i%keyCount)
					if i%10 == 0 {
						s.Set(key, "new_val")
					} else {
						s.Get(key)
					}
					i++
				}
			})
		})

		b.Run(fmt.Sprintf("%s/WriteHeavy", name), func(b *testing.B) {
			keyCount := 1000
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					key := fmt.Sprintf("key%d", i%keyCount)
					if i%2 == 0 {
						s.Set(key, "val")
					} else {
						s.Get(key)
					}
					i++
				}
			})
		})
	}
}
