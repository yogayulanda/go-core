package gateway

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/yogayulanda/go-core/config"
)

func TestEvaluateReadiness_DBRequiredFailed(t *testing.T) {
	cfg := &config.Config{
		Databases: map[string]config.DBConfig{
			"transaction": {Required: true},
		},
	}

	ok, report := evaluateReadiness(
		context.Background(),
		cfg,
		map[string]sqlPinger{
			"transaction": fakeSQLPinger{err: errors.New("down")},
		},
		nil,
		nil,
		nil,
	)
	if ok {
		t.Fatalf("expected not ready")
	}
	if report.Status != readinessStatusNotReady {
		t.Fatalf("expected status %q, got %q", readinessStatusNotReady, report.Status)
	}
	check, exists := report.Checks["database.transaction"]
	if !exists {
		t.Fatalf("expected database.transaction check")
	}
	if check.Status != checkStatusDown || !check.Required {
		t.Fatalf("unexpected database check: %+v", check)
	}
}

func TestEvaluateReadiness_DBOptionalIgnored(t *testing.T) {
	cfg := &config.Config{
		Databases: map[string]config.DBConfig{
			"history": {Required: false},
		},
	}

	ok, report := evaluateReadiness(
		context.Background(),
		cfg,
		map[string]sqlPinger{
			"history": fakeSQLPinger{err: errors.New("down")},
		},
		nil,
		nil,
		nil,
	)
	if !ok {
		t.Fatalf("expected ready")
	}
	check, exists := report.Checks["database.history"]
	if !exists {
		t.Fatalf("expected database.history check")
	}
	if check.Status != checkStatusSkipped || check.Required {
		t.Fatalf("unexpected optional db check: %+v", check)
	}
}

func TestEvaluateReadiness_RedisEnabledNilCache(t *testing.T) {
	cfg := &config.Config{
		Databases: map[string]config.DBConfig{},
		Redis:     config.RedisConfig{Enabled: true},
	}

	ok, report := evaluateReadiness(context.Background(), cfg, map[string]sqlPinger{}, nil, nil, nil)
	if ok {
		t.Fatalf("expected not ready")
	}
	check, exists := report.Checks["redis"]
	if !exists {
		t.Fatalf("expected redis check")
	}
	if check.Status != checkStatusDown || !check.Required {
		t.Fatalf("unexpected redis check: %+v", check)
	}
}

func TestEvaluateReadiness_MemcachedEnabledCacheFail(t *testing.T) {
	cfg := &config.Config{
		Databases: map[string]config.DBConfig{},
		Memcached: config.MemcachedConfig{Enabled: true},
	}

	ok, report := evaluateReadiness(
		context.Background(),
		cfg,
		map[string]sqlPinger{},
		nil,
		fakeCache{healthErr: errors.New("down")},
		nil,
	)
	if ok {
		t.Fatalf("expected not ready")
	}
	check, exists := report.Checks["memcached"]
	if !exists {
		t.Fatalf("expected memcached check")
	}
	if check.Status != checkStatusDown || !check.Required {
		t.Fatalf("unexpected memcached check: %+v", check)
	}
}

func TestEvaluateReadiness_KafkaFail(t *testing.T) {
	cfg := &config.Config{
		Databases: map[string]config.DBConfig{},
		Kafka: config.KafkaConfig{
			Enabled: true,
			Brokers: []string{"127.0.0.1:9092"},
		},
	}

	ok, report := evaluateReadiness(
		context.Background(),
		cfg,
		map[string]sqlPinger{},
		nil,
		nil,
		func(context.Context, []string) bool { return false },
	)
	if ok {
		t.Fatalf("expected not ready")
	}
	check, exists := report.Checks["kafka"]
	if !exists {
		t.Fatalf("expected kafka check")
	}
	if check.Status != checkStatusDown || !check.Required {
		t.Fatalf("unexpected kafka check: %+v", check)
	}
}

func TestEvaluateReadiness_AllReady(t *testing.T) {
	cfg := &config.Config{
		Databases: map[string]config.DBConfig{
			"transaction": {Required: true},
		},
		Redis:     config.RedisConfig{Enabled: true},
		Memcached: config.MemcachedConfig{Enabled: true},
		Kafka: config.KafkaConfig{
			Enabled: true,
			Brokers: []string{"127.0.0.1:9092"},
		},
	}

	ok, report := evaluateReadiness(
		context.Background(),
		cfg,
		map[string]sqlPinger{
			"transaction": fakeSQLPinger{},
		},
		fakeCache{},
		fakeCache{},
		func(context.Context, []string) bool { return true },
	)
	if !ok {
		t.Fatalf("expected ready")
	}
	if report.Status != readinessStatusReady {
		t.Fatalf("expected report status %q, got %q", readinessStatusReady, report.Status)
	}
	requiredChecks := []string{"database.transaction", "redis", "memcached", "kafka"}
	for _, key := range requiredChecks {
		check, exists := report.Checks[key]
		if !exists {
			t.Fatalf("missing check: %s", key)
		}
		if check.Status != checkStatusUp {
			t.Fatalf("expected check %s status up, got %+v", key, check)
		}
	}
}

func TestIsKafkaReady(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	defer ln.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if !isKafkaReady(ctx, []string{ln.Addr().String()}) {
		t.Fatalf("expected kafka ready for listening address")
	}

	if isKafkaReady(ctx, []string{"127.0.0.1:1"}) {
		t.Fatalf("expected kafka not ready for unreachable address")
	}
}

type fakeSQLPinger struct {
	err error
}

func (f fakeSQLPinger) PingContext(context.Context) error {
	return f.err
}

type fakeCache struct {
	healthErr error
}

func (f fakeCache) Get(context.Context, string) ([]byte, error) { return nil, nil }
func (f fakeCache) Set(context.Context, string, []byte, time.Duration) error {
	return nil
}
func (f fakeCache) Delete(context.Context, string) error { return nil }
func (f fakeCache) Health(context.Context) error         { return f.healthErr }
func (f fakeCache) Close() error                         { return nil }
