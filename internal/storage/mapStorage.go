package storage

import (
	"sync"
	"time"
)

// MapStorage is a thread-safe key-value storage.
type MapStorage struct {
	data    map[string]string // key - value
	expires map[string]int64  // key - expires time nanoseconds
	mu      sync.RWMutex
}

// NewMapStorage creates a new instance oа MapStorage.
func NewMapStorage() *MapStorage {
	return &MapStorage{
		data:    make(map[string]string),
		expires: make(map[string]int64),
		mu:      sync.RWMutex{},
	}
}

// Get returns the value and true if the key is found. Otherwise, "", false
func (m *MapStorage) Get(key string) (string, bool) {
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

// Set writes the value based on the options. Returns true if recording has been performed
func (m *MapStorage) Set(key, value string, options SetOptions) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, exists := m.data[key]
	if exists {
		exp, hasExp := m.expires[key]

		// key exists but is expired, clean it up now so logic below treats it as new
		if hasExp && time.Now().UnixNano() > exp {
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

// Delete deletes the key. Returns true if the key existed and was deleted
func (m *MapStorage) Delete(key string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.data[key]; ok {
		delete(m.data, key)
		delete(m.expires, key)
		return true
	}
	return false
}

// Expiry returns the remaining lifetime and status as ExpiryStatus
func (m *MapStorage) Expiry(key string) (time.Duration, ExpiryStatus) {
	m.mu.RLock()

	_, ok := m.data[key]
	exp, hasExp := m.expires[key]

	m.mu.RUnlock()

	// key does not exist
	if !ok {
		return 0, ExpNotFound
	}

	// key without TTL
	if !hasExp {
		return 0, ExpNoTimeout
	}

	now := time.Now().UnixNano()

	if now > exp {
		m.mu.Lock()
		defer m.mu.Unlock()

		if _, ok = m.data[key]; !ok {
			return 0, ExpNotFound
		}

		exp, hasExp = m.expires[key]
		if !hasExp {
			return 0, ExpNoTimeout
		}

		now = time.Now().UnixNano()

		// key expired
		if now > exp {
			delete(m.data, key)
			delete(m.expires, key)
			return 0, ExpNotFound
		}

		return time.Duration(exp - now), ExpActive
	}

	return time.Duration(exp - now), ExpActive
}

// Persist removes the expiration date of the key, making it eternal.
// Returns 1 if successful, 0 if the key was not found or had no TTL
func (m *MapStorage) Persist(key string) int64 {
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

// DeleteExpired randomly selects a limit of keys from each shard and delete if his TTL has expired
func (m *MapStorage) DeleteExpired(limit int) float64 {
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
