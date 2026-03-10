package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/yogayulanda/go-core/config"
)

type redisClient struct {
	client *redis.Client
}

func NewRedisFromConfig(cfg config.RedisConfig) (Cache, error) {

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Address,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Ping untuk fail-fast
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

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
