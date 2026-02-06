package storage

import (
	"io"
	"time"
)

type ExpiryStatus int

const (
	// ExpNotFound means that the key does not exist
	ExpNotFound ExpiryStatus = -2
	// ExpNoTimeout means that the key exists, but it does not have a TTL
	ExpNoTimeout ExpiryStatus = -1
	// ExpActive means that the key has an active lifetime
	ExpActive ExpiryStatus = 1
)

type SetOptions struct {
	TTL     time.Duration // key lifetime
	KeepTTL bool          // if true, retain the existing TTL (ignore TTL field)
	NX      bool          // only set if the key does not exist
	XX      bool          // only set if the key already exists
}

// Storage is a common interface for working with key-value storages
type Storage interface {
	// Get returns the value and true if the key is found. Otherwise, "", false
	Get(key string) (string, bool, error)

	// Set writes the value based on the options. Returns true if recording has been performed
	Set(key, value string, options SetOptions) bool

	// Delete deletes the key. Returns true if the key existed and was deleted
	Delete(key string) bool

	// Expiry returns the remaining lifetime and status as ExpiryStatus
	Expiry(key string) (time.Duration, ExpiryStatus)

	// Persist removes the expiration date of the key, making it eternal.
	// Returns 1 if successful, 0 if the key was not found or had no TTL
	Persist(key string) int64

	// DeleteExpired randomly selects a limit of keys from each shard and delete if his TTL has expired
	DeleteExpired(limit int) float64

	// Snapshot writes the entire state of the storage to the writer.
	// Implementation must ensure consistency (or shard-level consistency)
	Snapshot(w io.Writer) error

	// Restore reads the state from the reader and populates the storage
	Restore(r io.Reader) error

	// HSet sets the specified fields to their respective values in the hash stored at key
	HSet(key string, field, value []string) int64

	// HGet returns the value associated with field in the hash stored at key
	HGet(key, field string) (string, bool)
}
