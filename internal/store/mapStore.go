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

func (m *MapStore) Set(key, value string, ttl time.Duration) {
	m.mu.Lock()

	m.data[key] = value

	if ttl == 0 {
		// deleting the expiration date if it was earlier
		delete(m.expires, key)
	} else {
		m.expires[key] = time.Now().Add(ttl).UnixNano()
	}

	m.mu.Unlock()
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
