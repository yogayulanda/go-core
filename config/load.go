package config

import (
	"fmt"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Option func()

func WithDotEnv(path string) Option {
	return func() {
		loadDotEnv(path)
	}
}

func Load(opts ...Option) (*Config, error) {
	for _, opt := range opts {
		opt()
	}

	serviceName := strings.TrimSpace(getString("SERVICE_NAME", ""))
	dbs, err := loadDatabases()
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		App: AppConfig{
			ServiceName:     serviceName,
			Environment:     strings.TrimSpace(getString("APP_ENV", "")),
			LogLevel:        getString("LOG_LEVEL", "info"),
			ShutdownTimeout: 25 * time.Second,
		},
		Databases: dbs,
		Migration: MigrationConfig{
			AutoRun:     false,
			DBName:      "",
			Dir:         "",
			LockEnabled: true,
			LockKey:     serviceName,
			LockTimeout: 30 * time.Second,
		},
		GRPC: GRPCConfig{
			Port:        getInt("GRPC_PORT", 50051),
			TLSEnabled:  false,
			TLSCertFile: "",
			TLSKeyFile:  "",
		},
		HTTP: HTTPConfig{
			Port:               getInt("HTTP_PORT", 8080),
			TLSEnabled:         false,
			TLSCertFile:        "",
			TLSKeyFile:         "",
			PprofEnabled:       getBool("HTTP_PPROF_ENABLED", false),
			CORSAllowedOrigins: getStringSlice("HTTP_CORS_ALLOWED_ORIGINS"),
		},
		Observability: ObservabilityConfig{
			OTLPEndpoint:       getString("APM_ENDPOINT", ""),
			OTLPInsecure:       false,
			OTLPCACertFile:     "",
			TraceSamplingRatio: 0.1,
		},
		Log: LogConfig{
			MaskKeys:             []string{"password", "passwd", "secret", "token", "authorization", "apikey", "api_key", "pin", "otp", "cvv", "card", "private_key", "session", "phone", "msisdn", "account", "cif", "nik"},
			RequestStart:         false,
			SlowQueryThreshold:   time.Second,
			SlowRequestThreshold: 3 * time.Second,
		},
		Diagnostics: DiagnosticsConfig{Enabled: getBool("DIAG_ENABLED", true)},
		Auth: AuthConfig{
			Signature: SignatureConfig{
				Enabled:      strings.TrimSpace(getString("AUTH_SIGNATURE_MASTER_KEY", "")) != "",
				MasterKey:    getString("AUTH_SIGNATURE_MASTER_KEY", ""),
				HeaderKey:    "x-signature",
				TimestampKey: "x-timestamp",
				MaxTimeDrift: 5 * time.Minute,
			},
			InternalJWT: InternalJWTConfig{
				Enabled:             false,
				PublicKey:           "",
				JWKSEndpoint:        "",
				JWKSRefreshInterval: time.Hour,
				Issuer:              "",
				Audience:            "",
				Leeway:              30 * time.Second,
			},
		},
		Redis: RedisConfig{
			Enabled:  envIsSet("REDIS_ADDRESS"),
			Address:  getString("REDIS_ADDRESS", ""),
			Password: getString("REDIS_PASSWORD", ""),
			DB:       0,
		},
		Memcached: loadMemcachedConfig(),
		Kafka: KafkaConfig{
			Enabled:       envIsSet("KAFKA_BROKERS"),
			Brokers:       getStringSlice("KAFKA_BROKERS"),
			ClientID:      serviceName,
			Username:      getString("KAFKA_USERNAME", ""),
			Password:      getString("KAFKA_PASSWORD", ""),
			SASLMechanism: getString("KAFKA_SASL_MECHANISM", ""),
			JksFile:       getString("KAFKA_JKS_FILE", ""),
			JksPassword:   getString("KAFKA_JKS_PASSWORD", ""),
		},
		IgnoredEnvVars: ignoredEnvironmentVariableNames(),
	}

	return cfg, nil
}

func loadMemcachedConfig() MemcachedConfig {
	servers := getStringSlice("MEMCACHED_SERVERS")

	return MemcachedConfig{
		Enabled: envIsSet("MEMCACHED_SERVERS"),
		Servers: servers,
		Timeout: 2 * time.Second,
	}
}

func loadDatabases() (map[string]DBConfig, error) {
	connections := getStringSlice("DB_LIST")

	dbs := make(map[string]DBConfig)
	for _, conn := range connections {
		conn = NormalizeDBAlias(conn)
		if conn == "" {
			continue
		}

		cfg := loadDatabaseByConnection(conn)
		if !isSupportedDBDriver(cfg.Driver) {
			return nil, fmt.Errorf("DB_%s_DRIVER must be sqlserver or mysql", strings.ToUpper(conn))
		}
		if cfg.DSN == "" {
			if err := validateComposeFields(conn, cfg); err != nil {
				return nil, err
			}
			dsn, err := composeDSN(cfg)
			if err != nil {
				return nil, err
			}
			cfg.DSN = dsn
		}

		dbs[conn] = cfg
	}

	return dbs, nil
}

func validateComposeFields(name string, cfg DBConfig) error {
	if cfg.Host == "" {
		return fmt.Errorf("DB_%s_HOST is required when DSN is not set", strings.ToUpper(name))
	}
	if cfg.Port <= 0 {
		return fmt.Errorf("DB_%s_PORT must be > 0 when DSN is not set", strings.ToUpper(name))
	}
	if cfg.Name == "" {
		return fmt.Errorf("DB_%s_NAME is required when DSN is not set", strings.ToUpper(name))
	}
	if cfg.User == "" {
		return fmt.Errorf("DB_%s_USER is required when DSN is not set", strings.ToUpper(name))
	}
	return nil
}

func loadDatabaseByConnection(name string) DBConfig {
	prefix := DatabaseEnvPrefix(name)

	return DBConfig{
		Driver:          strings.ToLower(getString(prefix+"DRIVER", "")),
		Host:            getString(prefix+"HOST", ""),
		Port:            getInt(prefix+"PORT", 0),
		Name:            getString(prefix+"NAME", ""),
		User:            getString(prefix+"USER", ""),
		Password:        getString(prefix+"PASSWORD", ""),
		Params:          getString(prefix+"PARAMS", ""),
		Required:        true,
		MaxOpenConns:    20,
		MaxIdleConns:    10,
		ConnMaxIdleTime: 2 * time.Minute,
		ConnMaxLifetime: 5 * time.Minute,
	}
}

func composeDSN(cfg DBConfig) (string, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.Driver)) {
	case "mysql":
		return composeMySQLDSN(cfg), nil
	case "sqlserver":
		return composeSQLServerDSN(cfg)
	default:
		return "", fmt.Errorf("unsupported DB driver %q (supported: sqlserver, mysql)", cfg.Driver)
	}
}

func isSupportedDBDriver(driver string) bool {
	switch strings.ToLower(strings.TrimSpace(driver)) {
	case "sqlserver", "mysql":
		return true
	default:
		return false
	}
}

func composeMySQLDSN(cfg DBConfig) string {
	auth := cfg.User
	if cfg.Password != "" {
		auth = auth + ":" + cfg.Password
	}

	dsn := fmt.Sprintf("%s@tcp(%s:%d)/%s", auth, cfg.Host, cfg.Port, cfg.Name)
	if cfg.Params != "" {
		dsn = dsn + "?" + strings.TrimPrefix(cfg.Params, "?")
	}
	return dsn
}

func composeSQLServerDSN(cfg DBConfig) (string, error) {
	u := &url.URL{
		Scheme: "sqlserver",
		User:   url.UserPassword(cfg.User, cfg.Password),
		Host:   fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
	}
	q := url.Values{}
	if cfg.Name != "" {
		q.Set("database", cfg.Name)
	}
	if cfg.Params != "" {
		extra, err := url.ParseQuery(strings.TrimPrefix(cfg.Params, "?"))
		if err != nil {
			return "", fmt.Errorf("invalid sqlserver params: %w", err)
		}
		for k, values := range extra {
			for _, v := range values {
				q.Add(k, v)
			}
		}
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func getString(key, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}

func getInt(key string, defaultVal int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return parsed
}

func getBool(key string, defaultVal bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	parsed, err := strconv.ParseBool(val)
	if err != nil {
		return defaultVal
	}
	return parsed
}

func getStringSlice(key string) []string {
	val := os.Getenv(key)
	if val == "" {
		return nil
	}
	parts := strings.Split(val, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func envIsSet(key string) bool {
	_, ok := os.LookupEnv(key)
	return ok
}

func ignoredEnvironmentVariableNames() []string {
	var names []string
	seen := make(map[string]struct{})
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if isRemovedEnvironmentVariable(name) {
			if _, ok := seen[name]; !ok {
				names = append(names, name)
				seen[name] = struct{}{}
			}
		}
	}
	sort.Strings(names)
	return names
}

func isRemovedEnvironmentVariable(name string) bool {
	switch name {
	case "LOG_TIMEZONE", "SHUTDOWN_TIMEOUT", "OTEL_EXPORTER_OTLP_ENDPOINT",
		"OTEL_EXPORTER_OTLP_INSECURE", "OTEL_EXPORTER_OTLP_CA_CERT_FILE", "TRACE_SAMPLING_RATIO",
		"REDIS_ENABLED", "REDIS_DB", "MEMCACHED_ENABLED", "MEMCACHED_ADDRESS", "MEMCACHED_TIMEOUT",
		"MEMCACHE_HOST", "MEMCACHE_PORT", "KAFKA_ENABLED", "KAFKA_CLIENT_ID",
		"AUTH_SIGNATURE_ENABLED", "AUTH_SIGNATURE_HEADER_KEY", "AUTH_SIGNATURE_TIMESTAMP_KEY",
		"AUTH_SIGNATURE_MAX_TIME_DRIFT":
		return true
	}
	for _, prefix := range []string{"GRPC_TLS_", "HTTP_TLS_", "MIGRATION_", "INTERNAL_JWT_"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	if strings.HasPrefix(name, "DB_") {
		for _, suffix := range []string{"_DSN", "_REQUIRED", "_MAX_OPEN_CONNS", "_MAX_IDLE_CONNS", "_CONN_MAX_IDLE_TIME", "_CONN_MAX_LIFETIME"} {
			if strings.HasSuffix(name, suffix) {
				return true
			}
		}
	}
	return false
}
