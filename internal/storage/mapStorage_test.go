package storage

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

func TestMapStorage_Concurrency(t *testing.T) {
	s := NewMapStorage()
	const workers = 100
	const opsPerWorker = 100000

	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func(workerID int) {
			defer wg.Done()
			r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(workerID)))

			for j := 0; j < opsPerWorker; j++ {
				key := fmt.Sprintf("key-%d", r.Intn(50))
				val := fmt.Sprintf("val-%d", j)

				op := r.Intn(3)
				switch op {
				case 0:
					s.Set(key, val, SetOptions{
						TTL:     0,
						KeepTTL: false,
						NX:      false,
						XX:      false,
					})
				case 1:
					s.Get(key)
				case 2:
					s.Delete(key)
				}
			}
		}(i)
	}

	wg.Wait()
}

func FuzzMapStorage(f *testing.F) {
	s := NewMapStorage()

	f.Add("key1", "val1")
	f.Add("special", "!@#$%^&*()")

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
