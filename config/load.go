package config

import (
	"fmt"
	"net/url"
	"os"
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

	dbs, err := loadDatabases()
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		App: AppConfig{
			ServiceName:     getString("SERVICE_NAME", ""),
			Environment:     getString("APP_ENV", "dev"),
			LogLevel:        getString("LOG_LEVEL", "info"),
			ShutdownTimeout: getDuration("SHUTDOWN_TIMEOUT", 10*time.Second),
		},
		Databases: dbs,
		Migration: MigrationConfig{
			AutoRun:     getBool("MIGRATION_AUTO_RUN", false),
			DBName:      NormalizeDBAlias(getString("MIGRATION_DB", "")),
			Dir:         strings.TrimSpace(getString("MIGRATION_DIR", "")),
			LockEnabled: getBool("MIGRATION_LOCK_ENABLED", true),
			LockKey:     getString("MIGRATION_LOCK_KEY", ""),
			LockTimeout: getDuration("MIGRATION_LOCK_TIMEOUT", 30*time.Second),
		},
		GRPC: GRPCConfig{
			Port:        getInt("GRPC_PORT", 50051),
			TLSEnabled:  getBool("GRPC_TLS_ENABLED", false),
			TLSCertFile: getString("GRPC_TLS_CERT_FILE", ""),
			TLSKeyFile:  getString("GRPC_TLS_KEY_FILE", ""),
		},
		HTTP: HTTPConfig{
			Port:         getInt("HTTP_PORT", 8080),
			TLSEnabled:   getBool("HTTP_TLS_ENABLED", false),
			TLSCertFile:  getString("HTTP_TLS_CERT_FILE", ""),
			TLSKeyFile:   getString("HTTP_TLS_KEY_FILE", ""),
			PprofEnabled: getBool("HTTP_PPROF_ENABLED", false),
		},
		Observability: ObservabilityConfig{
			OTLPEndpoint:       getString("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
			OTLPInsecure:       getBool("OTEL_EXPORTER_OTLP_INSECURE", false),
			OTLPCACertFile:     getString("OTEL_EXPORTER_OTLP_CA_CERT_FILE", ""),
			TraceSamplingRatio: getFloat("TRACE_SAMPLING_RATIO", 0.1),
		},
		Auth: AuthConfig{
			Signature: SignatureConfig{
				Enabled:      getBool("AUTH_SIGNATURE_ENABLED", false),
				MasterKey:    getString("AUTH_SIGNATURE_MASTER_KEY", ""),
				HeaderKey:    getString("AUTH_SIGNATURE_HEADER_KEY", "x-signature"),
				TimestampKey: getString("AUTH_SIGNATURE_TIMESTAMP_KEY", "x-timestamp"),
				MaxTimeDrift: getDuration("AUTH_SIGNATURE_MAX_TIME_DRIFT", 5*time.Minute),
			},
			InternalJWT: InternalJWTConfig{
				Enabled:             getBool("INTERNAL_JWT_ENABLED", false),
				PublicKey:           getString("INTERNAL_JWT_PUBLIC_KEY", ""),
				JWKSEndpoint:        getString("INTERNAL_JWT_JWKS_ENDPOINT", ""),
				JWKSRefreshInterval: getDuration("INTERNAL_JWT_JWKS_REFRESH_INTERVAL", 1*time.Hour),
				Issuer:              getString("INTERNAL_JWT_ISSUER", ""),
				Audience:            getString("INTERNAL_JWT_AUDIENCE", ""),
				Leeway:              getDuration("INTERNAL_JWT_LEEWAY", 30*time.Second),
				IncludeMethods:      getStringSlice("INTERNAL_JWT_INCLUDE_METHODS"),
				ExcludeMethods:      getStringSlice("INTERNAL_JWT_EXCLUDE_METHODS"),
			},
		},
		Redis: RedisConfig{
			Enabled:  getBool("REDIS_ENABLED", false),
			Address:  getString("REDIS_ADDRESS", ""),
			Password: getString("REDIS_PASSWORD", ""),
			DB:       getInt("REDIS_DB", 0),
		},
		Memcached: loadMemcachedConfig(),
		Kafka: KafkaConfig{
			Enabled:     getBool("KAFKA_ENABLED", false),
			Brokers:     getStringSlice("KAFKA_BROKERS"),
			ClientID:    getString("KAFKA_CLIENT_ID", ""),
			Username:    getString("KAFKA_USERNAME", ""),
			Password:    getString("KAFKA_PASSWORD", ""),
			JksFile:     getString("KAFKA_JKS_FILE", ""),
			JksPassword: getString("KAFKA_JKS_PASSWORD", ""),
		},
	}

	return cfg, nil
}

func loadMemcachedConfig() MemcachedConfig {
	servers := getStringSlice("MEMCACHED_SERVERS")
	if len(servers) == 0 {
		if address := getString("MEMCACHED_ADDRESS", ""); address != "" {
			servers = []string{address}
		} else if host := getString("MEMCACHE_HOST", ""); host != "" {
			port := getString("MEMCACHE_PORT", "11211")
			servers = []string{fmt.Sprintf("%s:%s", host, port)}
		}
	}

	return MemcachedConfig{
		Enabled: getBool("MEMCACHED_ENABLED", false),
		Servers: servers,
		Timeout: getDuration("MEMCACHED_TIMEOUT", 2*time.Second),
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
		DSN:             getString(prefix+"DSN", ""),
		Host:            getString(prefix+"HOST", ""),
		Port:            getInt(prefix+"PORT", 0),
		Name:            getString(prefix+"NAME", ""),
		User:            getString(prefix+"USER", ""),
		Password:        getString(prefix+"PASSWORD", ""),
		Params:          getString(prefix+"PARAMS", ""),
		Required:        getBool(prefix+"REQUIRED", true),
		MaxOpenConns:    getInt(prefix+"MAX_OPEN_CONNS", 20),
		MaxIdleConns:    getInt(prefix+"MAX_IDLE_CONNS", 10),
		ConnMaxIdleTime: getDuration(prefix+"CONN_MAX_IDLE_TIME", 2*time.Minute),
		ConnMaxLifetime: getDuration(prefix+"CONN_MAX_LIFETIME", 5*time.Minute),
	}
}

func composeDSN(cfg DBConfig) (string, error) {
	switch cfg.Driver {
	case "mysql":
		return composeMySQLDSN(cfg), nil
	case "postgres":
		return composePostgresDSN(cfg)
	case "sqlserver":
		return composeSQLServerDSN(cfg)
	default:
		return "", fmt.Errorf("unsupported DB driver: %s", cfg.Driver)
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

func composePostgresDSN(cfg DBConfig) (string, error) {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.User, cfg.Password),
		Host:   fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Path:   "/" + cfg.Name,
	}
	if cfg.Params != "" {
		q, err := url.ParseQuery(strings.TrimPrefix(cfg.Params, "?"))
		if err != nil {
			return "", fmt.Errorf("invalid postgres params: %w", err)
		}
		u.RawQuery = q.Encode()
	}
	return u.String(), nil
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

func getFloat(key string, defaultVal float64) float64 {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	parsed, err := strconv.ParseFloat(val, 64)
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

func getDuration(key string, defaultVal time.Duration) time.Duration {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	parsed, err := time.ParseDuration(val)
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
