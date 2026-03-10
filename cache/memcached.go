package cache

import (
	"context"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/yogayulanda/go-core/config"
)

type memcachedClient struct {
	client *memcache.Client
}

func NewMemcachedFromConfig(cfg config.MemcachedConfig) (Cache, error) {
	client := memcache.New(cfg.Servers...)
	client.Timeout = cfg.Timeout

	// Fail-fast check.
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()
	if err := healthCheckMemcached(ctx, client); err != nil {
		return nil, err
	}

	return &memcachedClient{client: client}, nil
}

func (m *memcachedClient) Get(ctx context.Context, key string) ([]byte, error) {
	_ = ctx
	item, err := m.client.Get(key)
	if err != nil {
		return nil, err
	}
	return item.Value, nil
}

func (m *memcachedClient) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	_ = ctx
	ttlSeconds := int32(ttl.Seconds())
	if ttlSeconds < 0 {
		ttlSeconds = 0
	}
	return m.client.Set(&memcache.Item{
		Key:        key,
		Value:      value,
		Expiration: ttlSeconds,
	})
}

func (m *memcachedClient) Delete(ctx context.Context, key string) error {
	_ = ctx
	err := m.client.Delete(key)
	if err == memcache.ErrCacheMiss {
		return nil
	}
	return err
}

func (m *memcachedClient) Health(ctx context.Context) error {
	return healthCheckMemcached(ctx, m.client)
}

func (m *memcachedClient) Close() error {
	return nil
}

func healthCheckMemcached(ctx context.Context, c *memcache.Client) error {
	done := make(chan error, 1)
	go func() {
		_, err := c.Get("__go_core_memcached_healthcheck__")
		if err == memcache.ErrCacheMiss || err == nil {
			done <- nil
			return
		}
		done <- err
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}
