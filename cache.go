package gocommonweb

import "time"

// Cache represent a key value store functionality
type Cache interface {
	Get(key string) (string, error)
	Has(key string) (bool, error)
	Put(key string, value string) error
	PutWithTTL(key string, value string, ttl time.Duration) error
	Remove(key string) error
	Flush() error
}
