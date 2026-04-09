package cache

import (
	"context"
	"errors"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/yogayulanda/go-core/config"
	"github.com/yogayulanda/go-core/logger"
)

type memcachedClient struct {
	client memcachedBackend
}

type memcachedBackend interface {
	Get(key string) (*memcache.Item, error)
	Set(item *memcache.Item) error
	Delete(key string) error
}

type memcachedPinger interface {
	Get(key string) (*memcache.Item, error)
}

var newMemcachedBackend = func(cfg config.MemcachedConfig) memcachedBackend {
	client := memcache.New(cfg.Servers...)
	client.Timeout = cfg.Timeout
	return client
}

func NewMemcachedFromConfig(cfg config.MemcachedConfig, log logger.Logger) (Cache, error) {
	startedAt := time.Now()
	client := newMemcachedBackend(cfg)

	// Fail-fast check.
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()
	if err := healthCheckMemcached(ctx, client); err != nil {
		logConnect(ctx, log, "memcached", startedAt, "failed", "health_check_failed", map[string]interface{}{
			"servers":    cfg.Servers,
			"timeout_ms": cfg.Timeout.Milliseconds(),
		})
		return nil, err
	}

	logConnect(ctx, log, "memcached", startedAt, "success", "", map[string]interface{}{
		"servers":    cfg.Servers,
		"timeout_ms": cfg.Timeout.Milliseconds(),
	})

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

func healthCheckMemcached(ctx context.Context, c memcachedPinger) error {
	done := make(chan error, 1)
	go func() {
		_, err := c.Get("__go_core_memcached_healthcheck__")
		if err == nil || errors.Is(err, memcache.ErrCacheMiss) {
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
