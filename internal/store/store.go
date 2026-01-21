package store

import "time"

type Storage interface {
	Get(key string) (string, bool)
	Set(key, value string, ttl time.Duration)
	Delete(key string) bool
	Expiry(key string) (time.Duration, int)
}
