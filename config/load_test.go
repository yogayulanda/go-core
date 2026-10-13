package config

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestLoad_DatabaseAliasWithUnderscoreAndIdleTime(t *testing.T) {
	t.Setenv("SERVICE_NAME", "svc")
	t.Setenv("APP_ENV", "dev")
	t.Setenv("DB_LIST", "transaction_history")
	t.Setenv("DB_TRANSACTION_HISTORY_DRIVER", "sqlserver")
	t.Setenv("DB_TRANSACTION_HISTORY_HOST", "localhost")
	t.Setenv("DB_TRANSACTION_HISTORY_PORT", "1433")
	t.Setenv("DB_TRANSACTION_HISTORY_NAME", "test")
	t.Setenv("DB_TRANSACTION_HISTORY_USER", "user")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	db, ok := cfg.Databases["transaction_history"]
	if !ok {
		t.Fatalf("expected normalized alias transaction_history")
	}
	if db.Driver != "sqlserver" {
		t.Fatalf("unexpected driver: %q", db.Driver)
	}
	if db.ConnMaxIdleTime != 2*time.Minute {
		t.Fatalf("unexpected conn max idle time: %v", db.ConnMaxIdleTime)
	}
	if db.DSN == "" {
		t.Fatal("expected DSN to be composed from DB fields")
	}
}

func TestLoad_MigrationDefaultsAreExplicit(t *testing.T) {
	t.Setenv("SERVICE_NAME", "svc")
	t.Setenv("APP_ENV", "dev")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Migration.DBName != "" {
		t.Fatalf("expected empty migration db default, got %q", cfg.Migration.DBName)
	}
	if cfg.Migration.Dir != "" {
		t.Fatalf("expected empty migration dir default, got %q", cfg.Migration.Dir)
	}
	if cfg.Migration.LockKey != "svc" {
		t.Fatalf("expected migration lock key to default to service name, got %q", cfg.Migration.LockKey)
	}
	if cfg.Log.RequestStart || cfg.Log.SlowQueryThreshold != time.Second || cfg.Log.SlowRequestThreshold != 3*time.Second {
		t.Fatalf("unexpected log defaults: %+v", cfg.Log)
	}
	if len(cfg.Log.MaskKeys) == 0 {
		t.Fatal("expected default log mask keys")
	}
}

func TestLoad_ActiveEnvironmentVariablesMapToConfig(t *testing.T) {
	tests := []struct {
		name  string
		value string
		check func(*Config) bool
	}{
		{"SERVICE_NAME", "svc-active", func(c *Config) bool { return c.App.ServiceName == "svc-active" }},
		{"APP_ENV", "sit", func(c *Config) bool { return c.App.Environment == "sit" }},
		{"LOG_LEVEL", "debug", func(c *Config) bool { return c.App.LogLevel == "debug" }},
		{"GRPC_PORT", "50052", func(c *Config) bool { return c.GRPC.Port == 50052 }},
		{"HTTP_PORT", "8081", func(c *Config) bool { return c.HTTP.Port == 8081 }},
		{"HTTP_CORS_ALLOWED_ORIGINS", "https://one.example,https://two.example", func(c *Config) bool {
			return strings.Join(c.HTTP.CORSAllowedOrigins, ",") == "https://one.example,https://two.example"
		}},
		{"AUTH_SIGNATURE_MASTER_KEY", "test-key", func(c *Config) bool { return c.Auth.Signature.Enabled && c.Auth.Signature.MasterKey == "test-key" }},
		{"DB_LIST", "primary,archive", func(c *Config) bool { return len(c.Databases) == 2 && c.Databases["archive"].Driver == "mysql" }},
		{"DB_PRIMARY_DRIVER", "sqlserver", func(c *Config) bool { return c.Databases["primary"].Driver == "sqlserver" }},
		{"DB_PRIMARY_HOST", "db.example", func(c *Config) bool { return c.Databases["primary"].Host == "db.example" }},
		{"DB_PRIMARY_PORT", "1433", func(c *Config) bool { return c.Databases["primary"].Port == 1433 }},
		{"DB_PRIMARY_NAME", "appdb", func(c *Config) bool { return c.Databases["primary"].Name == "appdb" }},
		{"DB_PRIMARY_USER", "appuser", func(c *Config) bool { return c.Databases["primary"].User == "appuser" }},
		{"DB_PRIMARY_PASSWORD", "db-test-secret", func(c *Config) bool { return c.Databases["primary"].Password == "db-test-secret" }},
		{"DB_PRIMARY_PARAMS", "encrypt=disable", func(c *Config) bool { return strings.Contains(c.Databases["primary"].DSN, "encrypt=disable") }},
		{"REDIS_ADDRESS", "redis.example:6379", func(c *Config) bool { return c.Redis.Enabled && c.Redis.Address == "redis.example:6379" }},
		{"REDIS_PASSWORD", "redis-test-secret", func(c *Config) bool { return c.Redis.Password == "redis-test-secret" }},
		{"MEMCACHED_SERVERS", "cache-a:11211,cache-b:11211", func(c *Config) bool { return c.Memcached.Enabled && len(c.Memcached.Servers) == 2 }},
		{"KAFKA_BROKERS", "kafka.example:9092", func(c *Config) bool { return c.Kafka.Enabled && len(c.Kafka.Brokers) == 1 }},
		{"KAFKA_USERNAME", "kafka-user", func(c *Config) bool { return c.Kafka.Username == "kafka-user" }},
		{"KAFKA_PASSWORD", "kafka-test-secret", func(c *Config) bool { return c.Kafka.Password == "kafka-test-secret" }},
		{"KAFKA_SASL_MECHANISM", "SCRAM-SHA-256", func(c *Config) bool { return c.Kafka.SASLMechanism == "SCRAM-SHA-256" }},
		{"KAFKA_JKS_FILE", "/tmp/test.jks", func(c *Config) bool { return c.Kafka.JksFile == "/tmp/test.jks" }},
		{"KAFKA_JKS_PASSWORD", "jks-test-secret", func(c *Config) bool { return c.Kafka.JksPassword == "jks-test-secret" }},
		{"APM_ENDPOINT", "apm.example:4317", func(c *Config) bool { return c.Observability.OTLPEndpoint == "apm.example:4317" }},
		{"DIAG_ENABLED", "false", func(c *Config) bool { return !c.Diagnostics.Enabled }},
		{"HTTP_PPROF_ENABLED", "true", func(c *Config) bool { return c.HTTP.PprofEnabled }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clearConfigEnvironment(t)
			t.Setenv("SERVICE_NAME", "base-service")
			t.Setenv("APP_ENV", "local")
			t.Setenv("DB_LIST", "primary")
			t.Setenv("DB_PRIMARY_DRIVER", "mysql")
			t.Setenv("DB_PRIMARY_HOST", "localhost")
			t.Setenv("DB_PRIMARY_PORT", "3306")
			t.Setenv("DB_PRIMARY_NAME", "base")
			t.Setenv("DB_PRIMARY_USER", "base")
			t.Setenv("DB_ARCHIVE_DRIVER", "mysql")
			t.Setenv("DB_ARCHIVE_HOST", "localhost")
			t.Setenv("DB_ARCHIVE_PORT", "3306")
			t.Setenv("DB_ARCHIVE_NAME", "archive")
			t.Setenv("DB_ARCHIVE_USER", "archive")
			t.Setenv(test.name, test.value)

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() error: %v", err)
			}
			if !test.check(cfg) {
				t.Fatalf("%s did not populate its expected Config field", test.name)
			}
		})
	}
}

func TestLoad_RemovedEnvironmentVariablesAreIgnoredAndListed(t *testing.T) {
	removed := []string{
		"LOG_TIMEZONE", "SHUTDOWN_TIMEOUT", "GRPC_TLS_ENABLED", "GRPC_TLS_CERT_FILE", "GRPC_TLS_KEY_FILE",
		"HTTP_TLS_ENABLED", "HTTP_TLS_CERT_FILE", "HTTP_TLS_KEY_FILE", "DB_PRIMARY_DSN", "DB_PRIMARY_REQUIRED",
		"DB_PRIMARY_MAX_OPEN_CONNS", "DB_PRIMARY_MAX_IDLE_CONNS", "DB_PRIMARY_CONN_MAX_IDLE_TIME", "DB_PRIMARY_CONN_MAX_LIFETIME",
		"REDIS_ENABLED", "REDIS_DB", "MEMCACHED_ENABLED", "MEMCACHED_ADDRESS", "MEMCACHED_TIMEOUT", "MEMCACHE_HOST", "MEMCACHE_PORT",
		"KAFKA_ENABLED", "KAFKA_CLIENT_ID", "MIGRATION_AUTO_RUN", "MIGRATION_DB", "MIGRATION_DIR", "MIGRATION_LOCK_ENABLED",
		"MIGRATION_LOCK_KEY", "MIGRATION_LOCK_TIMEOUT", "OTEL_EXPORTER_OTLP_ENDPOINT", "OTEL_EXPORTER_OTLP_INSECURE",
		"OTEL_EXPORTER_OTLP_CA_CERT_FILE", "TRACE_SAMPLING_RATIO", "INTERNAL_JWT_ENABLED", "INTERNAL_JWT_PUBLIC_KEY",
		"INTERNAL_JWT_JWKS_ENDPOINT", "INTERNAL_JWT_JWKS_REFRESH_INTERVAL", "INTERNAL_JWT_ISSUER", "INTERNAL_JWT_AUDIENCE",
		"INTERNAL_JWT_LEEWAY", "INTERNAL_JWT_INCLUDE_METHODS", "INTERNAL_JWT_EXCLUDE_METHODS", "AUTH_SIGNATURE_ENABLED",
		"AUTH_SIGNATURE_HEADER_KEY", "AUTH_SIGNATURE_TIMESTAMP_KEY", "AUTH_SIGNATURE_MAX_TIME_DRIFT",
	}
	clearConfigEnvironment(t)
	t.Setenv("SERVICE_NAME", "base-service")
	t.Setenv("APP_ENV", "dev")
	for _, name := range removed {
		t.Setenv(name, "legacy-test-value")
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(cfg.IgnoredEnvVars) != len(removed) {
		t.Fatalf("expected %d removed names, got %d", len(removed), len(cfg.IgnoredEnvVars))
	}
	for _, name := range removed {
		if !contains(cfg.IgnoredEnvVars, name) {
			t.Errorf("removed variable %s was not reported", name)
		}
	}
	if cfg.App.ShutdownTimeout != 25*time.Second || cfg.GRPC.TLSEnabled || cfg.HTTP.TLSEnabled {
		t.Fatal("removed env variables changed current Config defaults")
	}
	if cfg.Redis.Enabled || cfg.Memcached.Enabled || cfg.Kafka.Enabled || cfg.Observability.OTLPEndpoint != "" {
		t.Fatal("removed env variables enabled an optional integration")
	}
	if cfg.Auth.InternalJWT.Enabled || cfg.Auth.Signature.Enabled || cfg.Redis.DB != 0 {
		t.Fatal("removed env variables changed auth or Redis Config defaults")
	}
}

func TestLoad_MinimalConfigValidAndOptionalIntegrationsDisabled(t *testing.T) {
	clearConfigEnvironment(t)
	t.Setenv("SERVICE_NAME", "minimal-service")
	t.Setenv("APP_ENV", "local")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("minimal config should validate: %v", err)
	}
	if cfg.Redis.Enabled || cfg.Memcached.Enabled || cfg.Kafka.Enabled || cfg.Observability.OTLPEndpoint != "" {
		t.Fatal("optional integrations should be disabled")
	}
}

func clearConfigEnvironment(t *testing.T) {
	t.Helper()
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if !strings.Contains(name, "_") && name != "SERVICE_NAME" && name != "APP_ENV" && name != "LOG_LEVEL" {
			continue
		}
		value, hadValue := os.LookupEnv(name)
		if err := os.Unsetenv(name); err != nil {
			t.Fatalf("unset %s: %v", name, err)
		}
		t.Cleanup(func() {
			if hadValue {
				_ = os.Setenv(name, value)
			} else {
				_ = os.Unsetenv(name)
			}
		})
	}
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
