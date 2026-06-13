package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	client  *redis.Client
	enabled bool
}

func NewRedisClient(redisURL string) *RedisClient {
	if redisURL == "" {
		slog.Info("Redis not configured, cache layer disabled")
		return &RedisClient{enabled: false}
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		slog.Warn("Invalid Redis URL, cache layer disabled", "error", err)
		return &RedisClient{enabled: false}
	}

	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		slog.Warn("Redis connection failed, cache layer disabled", "error", err)
		client.Close()
		return &RedisClient{enabled: false}
	}

	slog.Info("Redis cache initialized", "addr", opts.Addr)
	return &RedisClient{client: client, enabled: true}
}

func (r *RedisClient) Enabled() bool {
	return r.enabled
}

func (r *RedisClient) Close() error {
	if r.enabled && r.client != nil {
		return r.client.Close()
	}
	return nil
}

func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if !r.enabled {
		return nil
	}
	return r.client.Set(ctx, key, value, ttl).Err()
}

func (r *RedisClient) SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if !r.enabled {
		return nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value for cache: %w", err)
	}
	return r.client.Set(ctx, key, string(data), ttl).Err()
}

func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	if !r.enabled {
		return "", redis.Nil
	}
	return r.client.Get(ctx, key).Result()
}

func (r *RedisClient) GetJSON(ctx context.Context, key string, dest interface{}) error {
	if !r.enabled {
		return redis.Nil
	}
	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(data), dest)
}

func (r *RedisClient) Delete(ctx context.Context, keys ...string) error {
	if !r.enabled {
		return nil
	}
	return r.client.Del(ctx, keys...).Err()
}

func (r *RedisClient) Exists(ctx context.Context, key string) (bool, error) {
	if !r.enabled {
		return false, nil
	}
	n, err := r.client.Exists(ctx, key).Result()
	return n > 0, err
}

func (r *RedisClient) Incr(ctx context.Context, key string) (int64, error) {
	if !r.enabled {
		return 0, nil
	}
	return r.client.Incr(ctx, key).Result()
}

func (r *RedisClient) Expire(ctx context.Context, key string, ttl time.Duration) error {
	if !r.enabled {
		return nil
	}
	return r.client.Expire(ctx, key, ttl).Err()
}

func (r *RedisClient) Client() *redis.Client {
	return r.client
}
