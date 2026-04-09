package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/yogayulanda/go-core/config"
	"github.com/yogayulanda/go-core/logger"
)

type redisClient struct {
	client redisBackend
}

type redisBackend interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Ping(ctx context.Context) *redis.StatusCmd
	Close() error
}

var newRedisBackend = func(cfg config.RedisConfig) redisBackend {
	return redis.NewClient(&redis.Options{
		Addr:     cfg.Address,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
}

func NewRedisFromConfig(cfg config.RedisConfig, log logger.Logger) (Cache, error) {
	startedAt := time.Now()
	rdb := newRedisBackend(cfg)

	// Ping untuk fail-fast
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		logConnect(ctx, log, "redis", startedAt, "failed", "ping_failed", map[string]interface{}{
			"address": cfg.Address,
			"db":      cfg.DB,
		})
		return nil, err
	}

	logConnect(ctx, log, "redis", startedAt, "success", "", map[string]interface{}{
		"address": cfg.Address,
		"db":      cfg.DB,
	})

	return &redisClient{client: rdb}, nil
}

func (r *redisClient) Get(ctx context.Context, key string) ([]byte, error) {
	return r.client.Get(ctx, key).Bytes()
}

func (r *redisClient) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

func (r *redisClient) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

func (r *redisClient) Health(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

func (r *redisClient) Close() error {
	return r.client.Close()
}
