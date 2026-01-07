package store

import "sync"

type Storage interface {
	Get(key string) (string, bool)
	Set(key, value string)
	Delete(key string)
}

// MemoryStore for MVP
type MemoryStore struct {
	data map[string]string
	mu   sync.RWMutex
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data: make(map[string]string),
		mu:   sync.RWMutex{},
	}
}

func (m *MemoryStore) Get(key string) (string, bool) {
	m.mu.RLock()
	val, ok := m.data[key]
	m.mu.RUnlock()
	return val, ok
}

func (m *MemoryStore) Set(key, value string) {
	m.mu.Lock()
	m.data[key] = value
	m.mu.Unlock()
}

func (m *MemoryStore) Delete(key string) {
	m.mu.Lock()
	delete(m.data, key)
	m.mu.Unlock()
}
