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

	val, exists := m.data[key]
	if exists {
		if val.Type != TypeString {
			return false
		}

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
		case TypeHash:
			// [Count][KeyLen][Key][ValLen][Val]...
			h := value.Value.(map[string]HashField)
			if err := binary.Write(w, binary.LittleEndian, uint32(len(h))); err != nil {
				return err
			}

			now := time.Now().UnixNano()

			for field, val := range h {
				if val.ExpireAt > 0 && now > val.ExpireAt {
					continue
				}

				if err := writeString(w, field); err != nil {
					return err
				}
				if err := writeString(w, val.Value); err != nil {
					return err
				}
				if err := binary.Write(w, binary.LittleEndian, val.ExpireAt); err != nil {
					return err
				}
			}

		case TypeList:
			//TODO List
		case TypeSet:
			//TODO Set
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
		case TypeHash:
			var count uint32
			if err := binary.Read(r, binary.LittleEndian, &count); err != nil {
				return err
			}

			h := make(map[string]HashField, count)

			for range count {
				field, err := readString(r)
				if err != nil {
					return err
				}

				val, err := readString(r)
				if err != nil {
					return err
				}

				var expireAt int64
				if err := binary.Read(r, binary.LittleEndian, &expireAt); err != nil {
					return err
				}

				h[field] = HashField{Value: val, ExpireAt: expireAt}
			}
			value = h

		case TypeList:
			//TODO List
		case TypeSet:
			//TODO Set
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

// Hash

// getHash safely obtains the hash and results in the desired type
func (m *MapStorage) getHash(key string) (map[string]HashField, bool) {
	entry, exists := m.data[key]
	if !exists || entry.Type != TypeHash || entry.Value == nil {
		return nil, false
	}
	return entry.Value.(map[string]HashField), true
}

// checkFieldLocked checks the TTL of the field. If it has expired, it deletes it
// returns the number of elements and the presence of the field
func (m *MapStorage) checkFieldLocked(hash map[string]HashField, field string) (int, bool) {
	val, ok := hash[field]
	if !ok {
		return 0, false
	}

	if val.ExpireAt > 0 && time.Now().UnixNano() > val.ExpireAt {
		delete(hash, field)
		return len(hash), false
	}
	return len(hash), true
}

// HSet sets the specified fields to their respective values in the hash stored at key
func (m *MapStorage) HSet(key string, fields map[string]string) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	entity, ok := m.data[key]
	if ok && entity.Type != TypeHash {
		return -1 // wrong type
	}

	var hash map[string]HashField
	if !ok || entity.Value == nil {
		hash = make(map[string]HashField)
		m.data[key] = Entity{
			Type:  TypeHash,
			Value: hash,
		}
	} else {
		hash = entity.Value.(map[string]HashField)
	}

	var created int64 = 0

	for f, v := range fields {
		// when updating, the TTL value is reset
		if _, ok = hash[f]; !ok {
			created++
		}
		hash[f] = HashField{Value: v, ExpireAt: 0}
	}

	return created
}

// HGet returns the value associated with field in the hash stored at key
func (m *MapStorage) HGet(key, field string) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	hash, ok := m.getHash(key)
	if !ok {
		return "", false
	}

	lenHash, ok := m.checkFieldLocked(hash, field)
	if lenHash == 0 {
		delete(m.data, key)
		return "", false
	}

	if !ok {
		return "", false
	}

	return hash[field].Value, true
}

// HGetAll returns all fields and values of the hash stored at key
func (m *MapStorage) HGetAll(key string) map[string]string {
	m.mu.Lock()
	defer m.mu.Unlock()

	hash, ok := m.getHash(key)
	if !ok {
		return nil
	}

	result := make(map[string]string, len(hash))
	now := time.Now().UnixNano()

	for f, v := range hash {
		if v.ExpireAt > 0 && now > v.ExpireAt {
			delete(hash, f)
			continue
		}

		result[f] = v.Value
	}

	if len(hash) == 0 {
		delete(m.data, key)
		return nil
	}

	return result
}

// HDel removes the specified fields from the map stored at key
func (m *MapStorage) HDel(key string, fields []string) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	hash, ok := m.getHash(key)
	if !ok {
		return 0
	}

	var deleted int64

	for _, f := range fields {
		// skip field if its does not exist
		if _, ok := hash[f]; ok {
			delete(hash, f)
			deleted++
		}
	}

	if len(hash) == 0 {
		delete(m.data, key)
		delete(m.expires, key)
	}

	return deleted
}

// HExists returns 1 if field exist, 0 otherwise
func (m *MapStorage) HExists(key, field string) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	hash, ok := m.getHash(key)
	if !ok {
		return 0
	}

	lenHash, ok := m.checkFieldLocked(hash, field)
	if lenHash == 0 {
		delete(m.data, key)
		return 0
	}

	if !ok {
		return 0
	}

	return 1
}

// HLen returns the number of fields contained in the hash stored at key
func (m *MapStorage) HLen(key string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	hash, ok := m.getHash(key)
	if !ok {
		return 0
	}

	now := time.Now().UnixNano()
	var cnt int64

	for _, v := range hash {
		if v.ExpireAt > now {
			continue
		}
		cnt++
	}

	return cnt
}

// HKeys returns all field names in the hash stored at key
func (m *MapStorage) HKeys(key string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	hash, ok := m.getHash(key)
	if !ok {
		return nil
	}

	now := time.Now().UnixNano()
	response := make([]string, 0, len(hash))

	for f, v := range hash {
		if v.ExpireAt > now {
			continue
		}
		response = append(response, f)
	}

	return response
}

// HVals returns all values in the hash stored at key
func (m *MapStorage) HVals(key string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	hash, ok := m.getHash(key)
	if !ok {
		return nil
	}

	now := time.Now().UnixNano()
	response := make([]string, 0, len(hash))

	for _, v := range hash {
		if v.ExpireAt > now {
			continue
		}
		response = append(response, v.Value)
	}

	return response
}
