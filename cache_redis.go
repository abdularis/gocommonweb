package gocommonweb

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

type cacheRedis struct {
	rds      *redis.Client
	baseName string
}

// NewCacheRedis create cache backed by redis
func NewCacheRedis(redisClient *redis.Client, appName string) Cache {
	return &cacheRedis{
		rds:      redisClient,
		baseName: appName,
	}
}

func (c *cacheRedis) createKey(key string) string {
	return fmt.Sprintf("cache:%s:%s", c.baseName, key)
}

func (c *cacheRedis) Get(key string) (string, error) {
	return c.rds.Get(context.Background(), c.createKey(key)).Result()
}

func (c *cacheRedis) Has(key string) (bool, error) {
	res, err := c.rds.Exists(context.Background(), c.createKey(key)).Result()
	return res == 1, err
}

func (c *cacheRedis) Put(key string, value string) error {
	return c.rds.Set(context.Background(), c.createKey(key), value, 0).Err()
}

func (c *cacheRedis) PutWithTTL(key string, value string, ttl time.Duration) error {
	return c.rds.Set(context.Background(), c.createKey(key), value, ttl).Err()
}

func (c *cacheRedis) Remove(key string) error {
	return c.rds.Del(context.Background(), c.createKey(key)).Err()
}

func (c *cacheRedis) Flush() error {
	keyPattern := fmt.Sprintf("cache:%s:*", c.baseName)
	var cursor uint64 = 0
	for {
		keys, nextCursor, err := c.rds.Scan(context.Background(), cursor, keyPattern, 50).Result()
		if err != nil {
			return err
		}

		if len(keys) > 0 {
			_, err = c.rds.Del(context.Background(), keys...).Result()
			if err != nil {
				return err
			}
		}

		cursor = nextCursor
		if nextCursor <= 0 {
			logrus.Debug("[redis cache] scan iteration end, all flushed")
			break
		}
	}
	return nil
}
