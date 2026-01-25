package store

import (
	"sync"
	"time"
)

type MapStore struct {
	data    map[string]string // key - value
	expires map[string]int64  // key - expires time nanoseconds
	mu      sync.RWMutex
}

func NewMapStore() *MapStore {
	return &MapStore{
		data:    make(map[string]string),
		expires: make(map[string]int64),
		mu:      sync.RWMutex{},
	}
}

func (m *MapStore) Get(key string) (string, bool) {
	m.mu.RLock()
	exp, hasExp := m.expires[key]
	val, ok := m.data[key]
	m.mu.RUnlock()

	if !ok {
		return "", false
	}

	if hasExp && time.Now().UnixNano() > exp {
		m.mu.Lock()
		defer m.mu.Unlock()

		// checking again, can be changed while waiting for the lock
		exp, hasExp = m.expires[key]
		if hasExp && time.Now().UnixNano() > exp {
			delete(m.data, key)
			delete(m.expires, key)
			return "", false
		}

		if val, ok = m.data[key]; ok {
			return val, true
		}
		return "", false
	}

	return val, true
}

func (m *MapStore) Set(key, value string, options SetOptions) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, exists := m.data[key]
	if exists {
		exp, hasExp := m.expires[key]
		if hasExp && time.Now().UnixNano() > exp {
			// key exists but is expired, clean it up now so logic below treats it as new
			delete(m.data, key)
			delete(m.expires, key)
			exists = false
		}
	}

	if options.NX && exists {
		return false
	}

	if options.XX && !exists {
		return false
	}

	m.data[key] = value

	if options.KeepTTL {
		// if KEEPTTL is set, we do nothing to m.expires (retain existing)
		// however, if the key is new (freshly created), KEEPTTL behaves like no TTL
		if !exists {
			delete(m.expires, key)
		}
	} else {
		if options.TTL == 0 {
			// no TTL provided (and not KEEPTTL), so we remove any existing expiration (persist)
			delete(m.expires, key)
		} else {
			m.expires[key] = time.Now().Add(options.TTL).UnixNano()
		}
	}

	return true
}

func (m *MapStore) Delete(key string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.data[key]; ok {
		delete(m.data, key)
		delete(m.expires, key)
		return true
	}
	return false
}

// Expiry returns the remaining lifetime and status code in the second return value.
//
// The key does not exist (or expired)(-2).
// The key exists but has no expiration(-1).
// The key exists and has an expiration date (ttl is returned in time.Duration)(1)
func (m *MapStore) Expiry(key string) (time.Duration, int) {
	m.mu.RLock()

	_, ok := m.data[key]
	exp, hasExp := m.expires[key]

	m.mu.RUnlock()

	// key does not exist
	if !ok {
		return 0, -2
	}

	// key without TTL
	if !hasExp {
		return 0, -1
	}

	now := time.Now().UnixNano()

	if now > exp {
		m.mu.Lock()
		defer m.mu.Unlock()

		if _, ok = m.data[key]; !ok {
			return 0, -2
		}

		exp, hasExp = m.expires[key]
		if !hasExp {
			return 0, -1
		}

		now = time.Now().UnixNano()

		// key expired
		if now > exp {
			delete(m.data, key)
			delete(m.expires, key)
			return 0, -2
		}

		return time.Duration(exp - now), 1
	}

	return time.Duration(exp - now), 1
}

func (m *MapStore) Persist(key string) int64 {
	m.mu.RLock()

	_, ok := m.data[key]
	_, hasExp := m.expires[key]

	m.mu.RUnlock()

	if !ok || !hasExp {
		return 0
	}

	m.mu.Lock()

	_, ok = m.data[key]
	_, hasExp = m.expires[key]

	if !ok || !hasExp {
		return 0
	}

	delete(m.expires, key)

	m.mu.Unlock()

	return 1
}

func (m *MapStore) DeleteExpired(limit int) float64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.expires) == 0 {
		return 0.0
	}

	checked := 0
	expired := 0
	now := time.Now().UnixNano()

	// go map iteration is randomized by design
	for key, expTime := range m.expires {
		checked++
		if now > expTime {
			delete(m.data, key)
			delete(m.expires, key)
			expired++
		}

		if checked >= limit {
			break
		}
	}

	return float64(expired) / float64(checked)
}
