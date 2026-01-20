package store

import "sync"

type MapStore struct {
	data map[string]string
	mu   sync.RWMutex
}

func NewMapStore() *MapStore {
	return &MapStore{
		data: make(map[string]string),
		mu:   sync.RWMutex{},
	}
}

func (m *MapStore) Get(key string) (string, bool) {
	m.mu.RLock()
	val, ok := m.data[key]
	m.mu.RUnlock()
	return val, ok
}

func (m *MapStore) Set(key, value string) {
	m.mu.Lock()
	m.data[key] = value
	m.mu.Unlock()
}

func (m *MapStore) Delete(key string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.data[key]; ok {
		delete(m.data, key)
		return true
	}
	return false
}
