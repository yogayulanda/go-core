package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/yogayulanda/go-core/cache"
	"github.com/yogayulanda/go-core/config"
	"github.com/yogayulanda/go-core/database"
	"github.com/yogayulanda/go-core/logger"
	"github.com/yogayulanda/go-core/messaging"
	"github.com/yogayulanda/go-core/observability"
)

var newDatabaseFn = database.New
var ErrKafkaDisabled = errors.New("kafka disabled")

type App struct {
	cfg       *config.Config
	logger    logger.Logger
	metrics   *observability.Metrics
	lifecycle *Lifecycle

	dbs            map[string]*database.DB
	redisCache     cache.Cache
	memcachedCache cache.Cache
}

// New initializes core application dependencies.
// It does NOT start Kafka, consumer, or outbox automatically.
// Service layer decides messaging policy.
func New(ctx context.Context, cfg *config.Config) (*App, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	// 1. Logger
	log, err := logger.New(cfg.App.ServiceName, cfg.App.LogLevel)
	if err != nil {
		return nil, fmt.Errorf("init logger failed: %w", err)
	}

	// 2. Metrics
	metrics := observability.NewMetrics()

	// 3. Lifecycle
	lifecycle := NewLifecycle(cfg.App.ShutdownTimeout, log)

	// 4. Tracing (optional)
	shutdownTracer, err := observability.InitTracing(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("init tracing failed: %w", err)
	}
	lifecycle.Register(shutdownTracer)

	// 5. Databases
	dbs := make(map[string]*database.DB)
	for rawName, dbCfg := range cfg.Databases {
		name := config.NormalizeDBAlias(rawName)
		db, err := newDatabaseFn(dbCfg, log)
		if err != nil {
			if dbCfg.Required {
				return nil, fmt.Errorf("init database[%s] failed: %w", name, err)
			}

			log.Warn(ctx, "optional database unavailable; continuing startup",
				logger.Field{Key: "db_name", Value: name},
				logger.Field{Key: "driver", Value: dbCfg.Driver},
				logger.Field{Key: "reason", Value: err.Error()},
			)
			continue
		}

		dbs[name] = db
		lifecycle.Register(func(ctx context.Context) error {
			return db.Close()
		})
	}

	// 6. Redis (optional)
	var redisClient cache.Cache
	if cfg.Redis.Enabled {
		client, err := cache.NewRedisFromConfig(cfg.Redis)
		if err != nil {
			return nil, fmt.Errorf("init redis failed: %w", err)
		}
		redisClient = client

		lifecycle.Register(func(ctx context.Context) error {
			return client.Close()
		})
	}

	// 7. Memcached (optional)
	var memcachedClient cache.Cache
	if cfg.Memcached.Enabled {
		client, err := cache.NewMemcachedFromConfig(cfg.Memcached)
		if err != nil {
			return nil, fmt.Errorf("init memcached failed: %w", err)
		}
		memcachedClient = client

		lifecycle.Register(func(ctx context.Context) error {
			return client.Close()
		})
	}

	return &App{
		cfg:            cfg,
		logger:         log,
		metrics:        metrics,
		lifecycle:      lifecycle,
		dbs:            dbs,
		redisCache:     redisClient,
		memcachedCache: memcachedClient,
	}, nil
}

//
// ===== Getters =====
//

func (a *App) Config() *config.Config {
	return a.cfg
}

func (a *App) Logger() logger.Logger {
	return a.logger
}

func (a *App) Metrics() *observability.Metrics {
	return a.metrics
}

func (a *App) SQLByName(name string) *database.DB {
	return a.dbs[config.NormalizeDBAlias(name)]
}

func (a *App) SQLAll() map[string]*database.DB {
	out := make(map[string]*database.DB, len(a.dbs))
	for k, v := range a.dbs {
		out[k] = v
	}
	return out
}

// RedisCache returns Redis cache client if enabled.
func (a *App) RedisCache() cache.Cache {
	return a.redisCache
}

// MemcachedCache returns Memcached cache client if enabled.
func (a *App) MemcachedCache() cache.Cache {
	return a.memcachedCache
}

func (a *App) Lifecycle() *Lifecycle {
	return a.lifecycle
}

//
// ===== Messaging Helpers (Hybrid Pattern) =====
//

// NewKafkaPublisher creates a publisher and auto-registers close to lifecycle.
// No middleware is forced. Service decides policy.
func (a *App) NewKafkaPublisher(
	opts ...messaging.PublisherOption,
) (messaging.Publisher, error) {

	if !a.cfg.Kafka.Enabled {
		return nil, ErrKafkaDisabled
	}

	pub, err := messaging.NewKafkaPublisher(a.cfg.Kafka, opts...)
	if err != nil {
		return nil, err
	}

	a.lifecycle.Register(func(ctx context.Context) error {
		return pub.Close()
	})

	return pub, nil
}

// NewKafkaConsumer creates a consumer and auto-registers close to lifecycle.
// Service decides topic, group, retry, DLQ, concurrency.
func (a *App) NewKafkaConsumer(
	topic string,
	groupID string,
	handler messaging.Handler,
	opts ...messaging.ConsumerOption,
) (messaging.Consumer, error) {

	if !a.cfg.Kafka.Enabled {
		return nil, ErrKafkaDisabled
	}

	consumer, err := messaging.NewKafkaConsumer(
		a.cfg.Kafka,
		topic,
		groupID,
		handler,
		opts...,
	)
	if err != nil {
		return nil, err
	}

	a.lifecycle.Register(func(ctx context.Context) error {
		return consumer.Close()
	})

	return consumer, nil
}

//
// ===== Start & Shutdown =====
//

// Start blocks until context cancelled.
// Lifecycle shutdown will close DB, Redis, Publisher, Consumer, etc.
func (a *App) Start(ctx context.Context) error {
	a.logger.Info(ctx, "application starting")

	<-ctx.Done()

	a.logger.Info(ctx, "shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		a.cfg.App.ShutdownTimeout,
	)
	defer cancel()

	return a.lifecycle.Shutdown(shutdownCtx)
}
