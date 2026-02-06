package storage

import (
	"encoding/binary"
	"errors"
	"io"
	"sync"
	"time"
)

var (
	ErrWrongType = errors.New("WRONGTYPE")
)

// MapStorage is a thread-safe key-value storage.
type MapStorage struct {
	data    map[string]Entity // key - value
	expires map[string]int64  // key - expires time nanoseconds
	mu      sync.RWMutex
}

// NewMapStorage creates a new instance oа MapStorage.
func NewMapStorage() *MapStorage {
	return &MapStorage{
		data:    make(map[string]Entity),
		expires: make(map[string]int64),
		mu:      sync.RWMutex{},
	}
}

// Get returns the value and true if the key is found. Otherwise, "", false
func (m *MapStorage) Get(key string) (string, bool, error) {
	m.mu.RLock()
	exp, hasExp := m.expires[key]
	entity, ok := m.data[key]
	m.mu.RUnlock()

	if !ok {
		return "", false, nil
	}

	if entity.Type != TypeString {
		return "", false, ErrWrongType
	}

	if hasExp && time.Now().UnixNano() > exp {
		m.mu.Lock()
		defer m.mu.Unlock()

		// checking again, can be changed while waiting for the lock
		exp, hasExp = m.expires[key]
		if hasExp && time.Now().UnixNano() > exp {
			delete(m.data, key)
			delete(m.expires, key)
			return "", false, nil
		}

		entity, ok = m.data[key]
		if ok && entity.Type != TypeString {
			return "", false, ErrWrongType
		}
		if ok {
			return entity.Value.(string), true, nil
		}
		return "", false, nil
	}

	return entity.Value.(string), true, nil
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

	m.data[key] = Entity{
		Type:  TypeString,
		Value: value,
	}

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

// Expiry returns the remaining lifetime and status as expiryStatus
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

// writeString helper for writing a string with length
func writeString(w io.Writer, s string) error {
	lenBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(lenBuf, uint32(len(s)))
	if _, err := w.Write(lenBuf); err != nil {
		return err
	}
	if _, err := io.WriteString(w, s); err != nil {
		return err
	}
	return nil
}

// readString helper for reading string with length
func readString(r io.Reader) (string, error) {
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(r, lenBuf); err != nil {
		return "", err
	}
	strLen := binary.LittleEndian.Uint32(lenBuf)

	buf := make([]byte, strLen)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

// Snapshot serializes the shard data in Writer.
func (m *MapStorage) Snapshot(w io.Writer) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	header := make([]byte, 13)

	for key, value := range m.data {
		exp, hasExp := m.expires[key]
		if !hasExp {
			exp = 0
		}

		binary.LittleEndian.PutUint32(header[0:4], uint32(len(key)))
		binary.LittleEndian.PutUint64(header[4:12], uint64(exp))
		header[12] = byte(value.Type)

		// header
		if _, err := w.Write(header); err != nil {
			return err
		}

		// key
		if _, err := io.WriteString(w, key); err != nil {
			return err
		}

		// value
		switch value.Type {
		case TypeString:
			if err := writeString(w, value.Value.(string)); err != nil {
				return err
			}
		case TypeList:
			//TODO List
		case TypeSet:
			//TODO Set
		case TypeHash:
			//TODO Hash
		case TypeZSet:
			//TODO ZSet
		}

	}

	return nil
}

// Restore reads the stream and fills the map
func (m *MapStorage) Restore(r io.Reader) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	header := make([]byte, 13)

	for {
		_, err := io.ReadFull(r, header)
		if err == io.EOF {
			return nil // end of stream
		}
		if err != nil {
			return err
		}

		keyLen := binary.LittleEndian.Uint32(header[0:4])
		exp := int64(binary.LittleEndian.Uint64(header[4:12]))
		valueType := DataType(header[12])

		// read key
		keyBuf := make([]byte, keyLen)
		if _, err := io.ReadFull(r, keyBuf); err != nil {
			return err
		}
		key := string(keyBuf)

		// read value
		var value interface{}

		switch valueType {
		case TypeString:
			val, err := readString(r)
			if err != nil {
				return err
			}
			value = val
		case TypeList:
			//TODO List
		case TypeSet:
			//TODO Set
		case TypeHash:
			//TODO Hash
		case TypeZSet:
			//TODO ZSet
		}

		if exp > 0 && time.Now().UnixNano() > exp {
			continue
		}

		m.data[key] = Entity{
			Type:  valueType,
			Value: value,
		}
		if exp > 0 {
			m.expires[key] = exp
		}
	}
}

// HSet sets the specified fields to their respective values in the hash stored at key
func (m *MapStorage) HSet(key string, field, value []string) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	entity, ok := m.data[key]
	if ok && entity.Type != TypeHash {
		return -1 // wrong type
	}

	var hash map[string]string
	if !ok {
		hash = make(map[string]string)
		m.data[key] = Entity{
			Type:  TypeHash,
			Value: hash,
		}
	} else {
		hash = entity.Value.(map[string]string)
	}

	var created int64

	for i := 0; i != len(field); i++ {
		_, fieldExist := hash[field[i]]
		if !fieldExist {
			created++
		}
		hash[field[i]] = value[i]
	}

	return created
}

// HGet returns the value associated with field in the hash stored at key
func (m *MapStorage) HGet(key, field string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entity, exist := m.data[key]
	if !exist || entity.Type != TypeHash || entity.Value == nil {
		return "", false
	}

	value, ok := entity.Value.(map[string]string)[field]
	if !ok {
		return "", false
	}

	return value, true
}
