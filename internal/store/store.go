package store

import "time"

type SetOptions struct {
	TTL     time.Duration
	KeepTTL bool // if true, retain the existing TTL (ignore TTL field)
	NX      bool // only set if the key does not exist
	XX      bool // only set if the key already exists
}

type Storage interface {
	Get(key string) (string, bool)

	// Set returns true if the key was set, false if a condition (NX/XX) failed
	Set(key, value string, options SetOptions) bool

	Delete(key string) bool

	Expiry(key string) (time.Duration, int)

	Persist(key string) int64

	DeleteExpired(limit int) float64
}
