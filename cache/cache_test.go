package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/redis/go-redis/v9"
	"github.com/yogayulanda/go-core/config"
	"github.com/yogayulanda/go-core/logger"
)

func TestNewRedisFromConfig_Success_LogsStructuredConnect(t *testing.T) {
	restore := overrideRedisBackend(t, fakeRedisBackend{
		ping: redis.NewStatusResult("PONG", nil),
	})
	defer restore()

	log := &captureLogger{}

	cache, err := NewRedisFromConfig(config.RedisConfig{
		Enabled: true,
		Address: "127.0.0.1:6379",
		DB:      2,
	}, log)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if cache == nil {
		t.Fatalf("expected cache instance")
	}
	assertSingleCacheConnectLog(t, log, "redis", "success", "")
	entry := log.serviceLogs[0]
	if got := entry.Metadata["address"]; got != "127.0.0.1:6379" {
		t.Fatalf("expected redis address metadata, got %v", got)
	}
	if got := entry.Metadata["db"]; got != 2 {
		t.Fatalf("expected redis db metadata, got %v", got)
	}
}

func TestNewRedisFromConfig_InvalidAddress_ReturnError(t *testing.T) {
	log := &captureLogger{}

	_, err := NewRedisFromConfig(config.RedisConfig{
		Enabled: true,
		Address: "127.0.0.1:0",
	}, log)
	if err == nil {
		t.Fatalf("expected error")
	}
	assertSingleCacheConnectLog(t, log, "redis", "failed", "ping_failed")
}

func TestNewRedisFromConfig_NilLogger_Success(t *testing.T) {
	restore := overrideRedisBackend(t, fakeRedisBackend{
		ping: redis.NewStatusResult("PONG", nil),
	})
	defer restore()

	cache, err := NewRedisFromConfig(config.RedisConfig{
		Enabled: true,
		Address: "127.0.0.1:6379",
	}, nil)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if cache == nil {
		t.Fatalf("expected cache instance")
	}
}

func TestNewMemcachedFromConfig_Success_LogsStructuredConnect(t *testing.T) {
	restore := overrideMemcachedBackend(t, fakeMemcachedBackend{
		getFn: func(key string) (*memcache.Item, error) {
			return nil, memcache.ErrCacheMiss
		},
	})
	defer restore()

	log := &captureLogger{}

	cache, err := NewMemcachedFromConfig(config.MemcachedConfig{
		Enabled: true,
		Servers: []string{"127.0.0.1:11211"},
		Timeout: 75 * time.Millisecond,
	}, log)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if cache == nil {
		t.Fatalf("expected cache instance")
	}
	assertSingleCacheConnectLog(t, log, "memcached", "success", "")
	entry := log.serviceLogs[0]
	if got := entry.Metadata["timeout_ms"]; got != int64(75) {
		t.Fatalf("expected timeout metadata, got %v", got)
	}
}

func TestHealthCheckMemcached_CacheMiss_IsHealthy(t *testing.T) {
	err := healthCheckMemcached(context.Background(), fakeMemcachedBackend{
		getFn: func(key string) (*memcache.Item, error) {
			return nil, memcache.ErrCacheMiss
		},
	})
	if err != nil {
		t.Fatalf("expected cache miss to be treated as healthy, got: %v", err)
	}
}

func TestHealthCheckMemcached_Failure_ReturnsError(t *testing.T) {
	wantErr := errors.New("memcached unavailable")

	err := healthCheckMemcached(context.Background(), fakeMemcachedBackend{
		getFn: func(key string) (*memcache.Item, error) {
			return nil, wantErr
		},
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected memcached error, got: %v", err)
	}
}

func TestNewMemcachedFromConfig_InvalidAddress_ReturnError(t *testing.T) {
	log := &captureLogger{}

	_, err := NewMemcachedFromConfig(config.MemcachedConfig{
		Enabled: true,
		Servers: []string{"127.0.0.1:1"},
		Timeout: 50 * time.Millisecond,
	}, log)
	if err == nil {
		t.Fatalf("expected error")
	}
	assertSingleCacheConnectLog(t, log, "memcached", "failed", "health_check_failed")
}

func TestNewMemcachedFromConfig_NilLogger_Success(t *testing.T) {
	restore := overrideMemcachedBackend(t, fakeMemcachedBackend{
		getFn: func(key string) (*memcache.Item, error) {
			return nil, memcache.ErrCacheMiss
		},
	})
	defer restore()

	cache, err := NewMemcachedFromConfig(config.MemcachedConfig{
		Enabled: true,
		Servers: []string{"127.0.0.1:11211"},
		Timeout: 50 * time.Millisecond,
	}, nil)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if cache == nil {
		t.Fatalf("expected cache instance")
	}
}

func assertSingleCacheConnectLog(t *testing.T, log *captureLogger, backend string, status string, errorCode string) {
	t.Helper()

	if len(log.serviceLogs) != 1 {
		t.Fatalf("expected 1 service log, got %d", len(log.serviceLogs))
	}
	entry := log.serviceLogs[0]
	if entry.Operation != "cache_connect" || entry.Status != status || entry.ErrorCode != errorCode {
		t.Fatalf("unexpected cache log: %+v", entry)
	}
	if got := entry.Metadata["backend"]; got != backend {
		t.Fatalf("expected backend %s, got %v", backend, got)
	}
	if got := entry.Metadata["dependency_type"]; got != "cache" {
		t.Fatalf("expected dependency_type cache, got %v", got)
	}
}

func overrideRedisBackend(t *testing.T, backend redisBackend) func() {
	t.Helper()

	orig := newRedisBackend
	newRedisBackend = func(config.RedisConfig) redisBackend {
		return backend
	}
	return func() {
		newRedisBackend = orig
	}
}

func overrideMemcachedBackend(t *testing.T, backend memcachedBackend) func() {
	t.Helper()

	orig := newMemcachedBackend
	newMemcachedBackend = func(config.MemcachedConfig) memcachedBackend {
		return backend
	}
	return func() {
		newMemcachedBackend = orig
	}
}

type captureLogger struct {
	serviceLogs []logger.ServiceLog
}

func (l *captureLogger) Info(context.Context, string, ...logger.Field)         {}
func (l *captureLogger) Error(context.Context, string, ...logger.Field)        {}
func (l *captureLogger) Debug(context.Context, string, ...logger.Field)        {}
func (l *captureLogger) Warn(context.Context, string, ...logger.Field)         {}
func (l *captureLogger) LogDB(context.Context, logger.DBLog)                   {}
func (l *captureLogger) LogEvent(context.Context, logger.EventLog)             {}
func (l *captureLogger) LogTransaction(context.Context, logger.TransactionLog) {}
func (l *captureLogger) WithComponent(string) logger.Logger                    { return l }
func (l *captureLogger) LogService(_ context.Context, s logger.ServiceLog) {
	l.serviceLogs = append(l.serviceLogs, s)
}

type fakeRedisBackend struct {
	get   *redis.StringCmd
	set   *redis.StatusCmd
	del   *redis.IntCmd
	ping  *redis.StatusCmd
	close error
}

func (f fakeRedisBackend) Get(context.Context, string) *redis.StringCmd {
	if f.get != nil {
		return f.get
	}
	return redis.NewStringResult("", nil)
}

func (f fakeRedisBackend) Set(context.Context, string, interface{}, time.Duration) *redis.StatusCmd {
	if f.set != nil {
		return f.set
	}
	return redis.NewStatusResult("OK", nil)
}

func (f fakeRedisBackend) Del(context.Context, ...string) *redis.IntCmd {
	if f.del != nil {
		return f.del
	}
	return redis.NewIntResult(1, nil)
}

func (f fakeRedisBackend) Ping(context.Context) *redis.StatusCmd {
	if f.ping != nil {
		return f.ping
	}
	return redis.NewStatusResult("PONG", nil)
}

func (f fakeRedisBackend) Close() error {
	return f.close
}

type fakeMemcachedBackend struct {
	getFn     func(key string) (*memcache.Item, error)
	setErr    error
	deleteErr error
}

func (f fakeMemcachedBackend) Get(key string) (*memcache.Item, error) {
	if f.getFn != nil {
		return f.getFn(key)
	}
	return nil, memcache.ErrCacheMiss
}

func (f fakeMemcachedBackend) Set(*memcache.Item) error {
	return f.setErr
}

func (f fakeMemcachedBackend) Delete(string) error {
	return f.deleteErr
}
