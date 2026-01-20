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

	if ttl > 0 {
		m.expires[key] = time.Now().Add(ttl).UnixNano()
	} else {
		// deleting the expiration date if it was earlier
		delete(m.expires, key)
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
