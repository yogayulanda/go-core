package gateway

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	"github.com/yogayulanda/go-core/app"
	"github.com/yogayulanda/go-core/cache"
	"github.com/yogayulanda/go-core/config"
	"github.com/yogayulanda/go-core/database"
)

type sqlPinger interface {
	PingContext(ctx context.Context) error
}

type readinessCheck struct {
	Status   string `json:"status"`
	Required bool   `json:"required"`
	Message  string `json:"message,omitempty"`
}

type readinessReport struct {
	Status string                    `json:"status"`
	Checks map[string]readinessCheck `json:"checks"`
}

const (
	readinessStatusReady    = "ready"
	readinessStatusNotReady = "not_ready"
	checkStatusUp           = "up"
	checkStatusDown         = "down"
	checkStatusSkipped      = "skipped"
)

func registerHealthEndpoints(mux *runtime.ServeMux, application *app.App) error {
	if err := mux.HandlePath("GET", "/health", func(w http.ResponseWriter, r *http.Request, _ map[string]string) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}); err != nil {
		return err
	}

	if err := mux.HandlePath("GET", "/ready", func(w http.ResponseWriter, r *http.Request, _ map[string]string) {
		ok, report := evaluateReadiness(
			r.Context(),
			application.Config(),
			toSQLPingers(application.SQLAll()),
			application.RedisCache(),
			application.MemcachedCache(),
			isKafkaReady,
		)
		if !ok {
			writeJSON(w, http.StatusServiceUnavailable, report)
			return
		}

		writeJSON(w, http.StatusOK, report)
	}); err != nil {
		return err
	}

	return nil
}

func toSQLPingers(in map[string]*database.DB) map[string]sqlPinger {
	out := make(map[string]sqlPinger, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func evaluateReadiness(
	ctx context.Context,
	cfg *config.Config,
	sqlDBs map[string]sqlPinger,
	redisCache cache.Cache,
	memcachedCache cache.Cache,
	kafkaReadyFn func(ctx context.Context, brokers []string) bool,
) (bool, readinessReport) {
	report := readinessReport{
		Status: readinessStatusReady,
		Checks: map[string]readinessCheck{},
	}

	isReady := true
	markDown := func(name string, required bool, msg string) {
		report.Checks[name] = readinessCheck{
			Status:   checkStatusDown,
			Required: required,
			Message:  msg,
		}
		if required {
			isReady = false
		}
	}

	for dbName, dbCfg := range cfg.Databases {
		checkKey := "database." + dbName
		sqlDB, ok := sqlDBs[dbName]
		if !ok || sqlDB == nil {
			if dbCfg.Required {
				markDown(checkKey, true, "dependency unavailable")
				continue
			}
			report.Checks[checkKey] = readinessCheck{
				Status:   checkStatusSkipped,
				Required: false,
				Message:  "optional dependency",
			}
			continue
		}

		if !dbCfg.Required {
			report.Checks[checkKey] = readinessCheck{
				Status:   checkStatusSkipped,
				Required: false,
				Message:  "optional dependency",
			}
			continue
		}

		if err := sqlDB.PingContext(ctx); err != nil {
			markDown(checkKey, true, "health check failed")
			continue
		}

		report.Checks[checkKey] = readinessCheck{
			Status:   checkStatusUp,
			Required: true,
		}
	}

	redisCheck := evaluateCacheDependency(ctx, cfg.Redis.Enabled, redisCache)
	report.Checks["redis"] = redisCheck
	if redisCheck.Status == checkStatusDown {
		isReady = false
	}

	memcachedCheck := evaluateCacheDependency(ctx, cfg.Memcached.Enabled, memcachedCache)
	report.Checks["memcached"] = memcachedCheck
	if memcachedCheck.Status == checkStatusDown {
		isReady = false
	}

	if cfg.Kafka.Enabled {
		if kafkaReadyFn == nil || !kafkaReadyFn(ctx, cfg.Kafka.Brokers) {
			markDown("kafka", true, "health check failed")
		} else {
			report.Checks["kafka"] = readinessCheck{
				Status:   checkStatusUp,
				Required: true,
			}
		}
	} else {
		report.Checks["kafka"] = readinessCheck{
			Status:   checkStatusSkipped,
			Required: false,
			Message:  "disabled",
		}
	}

	if !isReady {
		report.Status = readinessStatusNotReady
	}

	return isReady, report
}

func isKafkaReady(ctx context.Context, brokers []string) bool {
	if len(brokers) == 0 {
		return false
	}

	timeout := 2 * time.Second
	dialer := &net.Dialer{Timeout: timeout}

	for _, broker := range brokers {
		dialCtx, cancel := context.WithTimeout(ctx, timeout)
		conn, err := dialer.DialContext(dialCtx, "tcp", broker)
		cancel()
		if err != nil {
			continue
		}
		_ = conn.Close()
		return true
	}

	return false
}

func writeJSON(w http.ResponseWriter, statusCode int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(body)
}

func evaluateCacheDependency(ctx context.Context, enabled bool, dependency cache.Cache) readinessCheck {
	if !enabled {
		return readinessCheck{
			Status:   checkStatusSkipped,
			Required: false,
			Message:  "disabled",
		}
	}

	if dependency == nil {
		return readinessCheck{
			Status:   checkStatusDown,
			Required: true,
			Message:  "dependency unavailable",
		}
	}

	checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := dependency.Health(checkCtx); err != nil {
		return readinessCheck{
			Status:   checkStatusDown,
			Required: true,
			Message:  "health check failed",
		}
	}

	return readinessCheck{
		Status:   checkStatusUp,
		Required: true,
	}
}
