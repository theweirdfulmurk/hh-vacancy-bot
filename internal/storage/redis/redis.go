package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Cache represents redis client
type Cache struct {
	client *redis.Client
	logger *zap.Logger
}

func New(addr, password string, db int, logger *zap.Logger) (*Cache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 5,
	})

	// check connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	logger.Info("successfully connected to Redis")

	return &Cache{
		client: client,
		logger: logger,
	}, nil
}

func (c *Cache) Close() error {
	return c.client.Close()
}

func (c *Cache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// Set saves value to Redis with TTL
func (c *Cache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	err = c.client.Set(ctx, key, data, ttl).Err()
	if err != nil {
		c.logger.Error("failed to set cache",
			zap.String("key", key),
			zap.Error(err),
		)
		return fmt.Errorf("set cache: %w", err)
	}

	return nil
}

func (c *Cache) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return fmt.Errorf("key not found")
	}
	if err != nil {
		c.logger.Error("failed to get cache",
			zap.String("key", key),
			zap.Error(err),
		)
		return fmt.Errorf("get cache: %w", err)
	}

	err = json.Unmarshal(data, dest)
	if err != nil {
		return fmt.Errorf("failed to unmarshal value: %w", err)
	}

	return nil
}

func (c *Cache) Delete(ctx context.Context, key string) error {
	err := c.client.Del(ctx, key).Err()
	if err != nil {
		c.logger.Error("failed to delete cache",
			zap.String("key", key),
			zap.Error(err),
		)
		return fmt.Errorf("delete cache: %w", err)
	}

	return nil
}

func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	result, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		c.logger.Error("failed to check key existence",
			zap.String("key", key),
			zap.Error(err),
		)
		return false, fmt.Errorf("exists: %w", err)
	}

	return result > 0, nil
}

func (c *Cache) SetString(ctx context.Context, key, value string, ttl time.Duration) error {
	err := c.client.Set(ctx, key, value, ttl).Err()
	if err != nil {
		c.logger.Error("failed to set string",
			zap.String("key", key),
			zap.Error(err),
		)
		return fmt.Errorf("set string: %w", err)
	}

	return nil
}

func (c *Cache) GetString(ctx context.Context, key string) (string, error) {
	value, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("key not found")
	}
	if err != nil {
		c.logger.Error("failed to get string",
			zap.String("key", key),
			zap.Error(err),
		)
		return "", fmt.Errorf("get string: %w", err)
	}

	return value, nil
}

func (c *Cache) Increment(ctx context.Context, key string) (int64, error) {
	value, err := c.client.Incr(ctx, key).Result()
	if err != nil {
		c.logger.Error("failed to increment",
			zap.String("key", key),
			zap.Error(err),
		)
		return 0, fmt.Errorf("increment: %w", err)
	}

	return value, nil
}

// IncrementWithExpiry increments counter and sets TTL if the key is new
func (c *Cache) IncrementWithExpiry(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	pipe := c.client.Pipeline()
	incrCmd := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, ttl)

	_, err := pipe.Exec(ctx)
	if err != nil {
		c.logger.Error("failed to increment with expiry",
			zap.String("key", key),
			zap.Error(err),
		)
		return 0, fmt.Errorf("increment with expiry: %w", err)
	}

	return incrCmd.Val(), nil
}

func (c *Cache) GetInt(ctx context.Context, key string) (int64, error) {
	value, err := c.client.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		c.logger.Error("failed to get int",
			zap.String("key", key),
			zap.Error(err),
		)
		return 0, fmt.Errorf("get int: %w", err)
	}

	return value, nil
}

func (c *Cache) SetWithExpiry(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return c.Set(ctx, key, value, ttl)
}

func (c *Cache) Keys(ctx context.Context, pattern string) ([]string, error) {
	keys, err := c.client.Keys(ctx, pattern).Result()
	if err != nil {
		c.logger.Error("failed to get keys",
			zap.String("pattern", pattern),
			zap.Error(err),
		)
		return nil, fmt.Errorf("keys: %w", err)
	}

	return keys, nil
}

func (c *Cache) FlushAll(ctx context.Context) error {
	err := c.client.FlushAll(ctx).Err()
	if err != nil {
		c.logger.Error("failed to flush all", zap.Error(err))
		return fmt.Errorf("flush all: %w", err)
	}

	c.logger.Warn("Redis flushed - all data deleted")
	return nil
}