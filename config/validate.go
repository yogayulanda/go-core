package config

import (
	"net/url"
	"strings"
)

func (c *Config) Validate() error {
	issues := c.ValidateIssues()
	if len(issues) == 0 {
		return nil
	}
	return &ValidationErrors{Issues: issues}
}

func (c *Config) ValidateIssues() []ValidationIssue {
	var issues []ValidationIssue

	if c.App.ServiceName == "" {
		issues = append(issues, issue("app", "SERVICE_NAME", "SERVICE_NAME is required"))
	}
	if c.App.Environment == "" {
		issues = append(issues, issue("app", "APP_ENV", "APP_ENV is required"))
	} else {
		switch c.App.Environment {
		case "local", "dev", "sit", "uat", "prod":
		default:
			issues = append(issues, issue("app", "APP_ENV", "APP_ENV must be one of local, dev, sit, uat, prod"))
		}
	}
	if c.App.DrainDelay < 0 {
		issues = append(issues, issue("app", "DrainDelay", "DrainDelay must be >= 0"))
	}
	if c.App.ShutdownTimeout <= 0 {
		issues = append(issues, issue("app", "ShutdownTimeout", "ShutdownTimeout must be > 0"))
	}
	if c.Diagnostics.Enabled {
		if c.Diagnostics.HeapThresholdPercent < 1 || c.Diagnostics.HeapThresholdPercent > 100 {
			issues = append(issues, issue("diagnostics", "HeapThresholdPercent", "HeapThresholdPercent must be between 1 and 100 when diagnostics are enabled"))
		}
		if c.Diagnostics.GoroutineThreshold <= 0 {
			issues = append(issues, issue("diagnostics", "GoroutineThreshold", "GoroutineThreshold must be > 0 when diagnostics are enabled"))
		}
		if c.Diagnostics.Cooldown <= 0 || c.Diagnostics.CheckInterval <= 0 {
			issues = append(issues, issue("diagnostics", "CooldownAndCheckInterval", "diagnostic Cooldown and CheckInterval must be > 0 when diagnostics are enabled"))
		}
	}
	if c.Log.SlowQueryThreshold < 0 {
		issues = append(issues, issue("log", "SlowQueryThreshold", "SlowQueryThreshold must be >= 0"))
	}
	if c.Log.SlowRequestThreshold < 0 {
		issues = append(issues, issue("log", "SlowRequestThreshold", "SlowRequestThreshold must be >= 0"))
	}
	if c.HTTP.ReadHeaderTimeout <= 0 || c.HTTP.ReadTimeout <= 0 || c.HTTP.WriteTimeout <= 0 || c.HTTP.IdleTimeout <= 0 || c.HTTP.MaxBodyBytes <= 0 {
		issues = append(issues, issue("http", "TimeoutsAndMaxBodyBytes", "HTTP timeouts and MaxBodyBytes must be > 0"))
	}
	if c.HTTP.WriteTimeout <= c.HTTP.ReadTimeout {
		issues = append(issues, issue("http", "WriteTimeout", "HTTP WriteTimeout must be greater than ReadTimeout"))
	}
	if c.HTTP.TrustedProxyHops < 0 {
		issues = append(issues, issue("http", "TrustedProxyHops", "TrustedProxyHops must be >= 0"))
	}
	if c.GRPC.Port < 1 || c.GRPC.Port > 65535 {
		issues = append(issues, issue("grpc", "GRPC_PORT", "GRPC_PORT must be between 1 and 65535"))
	}
	if c.HTTP.Port < 1 || c.HTTP.Port > 65535 {
		issues = append(issues, issue("http", "HTTP_PORT", "HTTP_PORT must be between 1 and 65535"))
	}
	if c.GRPC.Port == c.HTTP.Port {
		issues = append(issues, issue("app", "GRPC_PORT/HTTP_PORT", "GRPC_PORT and HTTP_PORT must be different"))
	}
	if c.HTTP.PprofEnabled && (c.App.Environment == "prod" || c.App.Environment == "production") {
		issues = append(issues, issue("http", "HTTP_PPROF_ENABLED", "pprof is forbidden in production"))
	}

	for name, db := range c.Databases {
		if !isSupportedDBDriver(db.Driver) {
			issues = append(issues, issue("database", "DB_"+strings.ToUpper(name)+"_DRIVER", "DB_"+strings.ToUpper(name)+"_DRIVER must be sqlserver or mysql"))
		}
		if db.DSN == "" {
			issues = append(issues, issue("database", "DB_"+strings.ToUpper(name), "database host, port, name, and user are required"))
		}
		if db.MaxOpenConns <= 0 {
			issues = append(issues, issue("database", "DB_"+strings.ToUpper(name)+"_MAX_OPEN_CONNS", "DB_"+strings.ToUpper(name)+"_MAX_OPEN_CONNS must be > 0"))
		}
		if db.MaxIdleConns < 0 {
			issues = append(issues, issue("database", "DB_"+strings.ToUpper(name)+"_MAX_IDLE_CONNS", "DB_"+strings.ToUpper(name)+"_MAX_IDLE_CONNS must be >= 0"))
		}
		if db.ConnMaxIdleTime < 0 {
			issues = append(issues, issue("database", "DB_"+strings.ToUpper(name)+"_CONN_MAX_IDLE_TIME", "DB_"+strings.ToUpper(name)+"_CONN_MAX_IDLE_TIME must be >= 0"))
		}
		if db.Port < 0 || db.Port > 65535 || (db.Port == 0 && db.DSN == "") {
			issues = append(issues, issue("database", "DB_"+strings.ToUpper(name)+"_PORT", "DB_"+strings.ToUpper(name)+"_PORT must be between 1 and 65535 when set or when DSN is not set"))
		}
	}

	if c.Migration.AutoRun {
		migrationDB := NormalizeDBAlias(c.Migration.DBName)
		if migrationDB == "" {
			issues = append(issues, issue("migration", "MIGRATION_DB", "MIGRATION_DB is required when MIGRATION_AUTO_RUN=true"))
		} else if !hasDatabaseAlias(c.Databases, migrationDB) {
			issues = append(issues, issue("migration", "MIGRATION_DB", "MIGRATION_DB must exist in DB_LIST when MIGRATION_AUTO_RUN=true"))
		}

		if strings.TrimSpace(c.Migration.Dir) == "" {
			issues = append(issues, issue("migration", "MIGRATION_DIR", "MIGRATION_DIR is required when MIGRATION_AUTO_RUN=true"))
		}
		if c.Migration.LockEnabled && c.Migration.LockTimeout <= 0 {
			issues = append(issues, issue("migration", "MIGRATION_LOCK_TIMEOUT", "MIGRATION_LOCK_TIMEOUT must be > 0 when MIGRATION_LOCK_ENABLED=true"))
		}
	}

	if c.Kafka.Enabled && len(c.Kafka.Brokers) == 0 {
		issues = append(issues, issue("kafka", "KAFKA_BROKERS", "KAFKA_BROKERS must not be empty when Kafka is enabled"))
	}
	kafkaUsername := strings.TrimSpace(c.Kafka.Username)
	kafkaPassword := strings.TrimSpace(c.Kafka.Password)
	kafkaMechanism := strings.ToUpper(strings.TrimSpace(c.Kafka.SASLMechanism))
	if kafkaUsername != "" {
		if kafkaPassword == "" {
			issues = append(issues, issue("kafka", "KAFKA_PASSWORD", "KAFKA_PASSWORD is required when KAFKA_USERNAME is set"))
		}
		switch kafkaMechanism {
		case "PLAIN", "SCRAM-SHA-256", "SCRAM-SHA-512":
		default:
			issues = append(issues, issue("kafka", "KAFKA_SASL_MECHANISM", "KAFKA_SASL_MECHANISM must be PLAIN, SCRAM-SHA-256, or SCRAM-SHA-512 when KAFKA_USERNAME is set"))
		}
	} else if kafkaPassword != "" || kafkaMechanism != "" {
		issues = append(issues, issue("kafka", "KAFKA_USERNAME", "KAFKA_USERNAME is required when SASL credentials or mechanism are set"))
	}
	if c.Kafka.JksPassword != "" && strings.TrimSpace(c.Kafka.JksFile) == "" {
		issues = append(issues, issue("kafka", "KAFKA_JKS_FILE", "KAFKA_JKS_FILE is required when KAFKA_JKS_PASSWORD is set"))
	}

	if c.Observability.OTLPInsecure && strings.TrimSpace(c.Observability.OTLPCACertFile) != "" {
		issues = append(issues, issue("observability", "OTLPCACertFile", "OTLPCACertFile must be empty when OTLPInsecure is true"))
	}

	if c.Redis.Enabled && c.Redis.Address == "" {
		issues = append(issues, issue("redis", "REDIS_ADDRESS", "REDIS_ADDRESS is required when Redis is enabled"))
	}

	if c.Memcached.Enabled {
		if len(c.Memcached.Servers) == 0 {
			issues = append(issues, issue("memcached", "MEMCACHED_SERVERS", "MEMCACHED_SERVERS is required when Memcached is enabled"))
		}
		if c.Memcached.Timeout <= 0 {
			issues = append(issues, issue("memcached", "Timeout", "Memcached Timeout must be > 0 when enabled"))
		}
	}

	if c.Auth.InternalJWT.Enabled {
		hasPublicKey := strings.TrimSpace(c.Auth.InternalJWT.PublicKey) != ""
		jwksEndpoint := strings.TrimSpace(c.Auth.InternalJWT.JWKSEndpoint)
		hasJWKSEndpoint := jwksEndpoint != ""
		if hasPublicKey == hasJWKSEndpoint {
			message := "InternalJWT requires exactly one of PublicKey or JWKSEndpoint"
			if hasPublicKey {
				message += "; both are set; remove one"
			}
			issues = append(issues, issue("auth", "InternalJWT.KeySource", message))
		}
		if hasJWKSEndpoint && !validJWKSURL(jwksEndpoint, c.App.Environment) {
			issues = append(issues, issue("auth", "InternalJWT.JWKSEndpoint", "InternalJWT JWKSEndpoint must be an https URL (http is allowed only in local)"))
		}
	}
	if c.Auth.InternalJWT.Enabled && c.Auth.InternalJWT.Leeway < 0 {
		issues = append(issues, issue("auth", "InternalJWT.Leeway", "InternalJWT Leeway must be >= 0 when enabled"))
	}
	if c.Auth.InternalJWT.Enabled &&
		len(c.Auth.InternalJWT.IncludeMethods) > 0 &&
		len(c.Auth.InternalJWT.ExcludeMethods) > 0 {
		issues = append(issues, issue("auth", "InternalJWT.IncludeMethods", "InternalJWT IncludeMethods and ExcludeMethods cannot be used together"))
	}

	if c.GRPC.TLSEnabled {
		if strings.TrimSpace(c.GRPC.TLSCertFile) == "" {
			issues = append(issues, issue("grpc", "TLSCertFile", "GRPC TLSCertFile is required when TLS is enabled"))
		}
		if strings.TrimSpace(c.GRPC.TLSKeyFile) == "" {
			issues = append(issues, issue("grpc", "TLSKeyFile", "GRPC TLSKeyFile is required when TLS is enabled"))
		}
	}

	if c.HTTP.TLSEnabled {
		if strings.TrimSpace(c.HTTP.TLSCertFile) == "" {
			issues = append(issues, issue("http", "TLSCertFile", "HTTP TLSCertFile is required when TLS is enabled"))
		}
		if strings.TrimSpace(c.HTTP.TLSKeyFile) == "" {
			issues = append(issues, issue("http", "TLSKeyFile", "HTTP TLSKeyFile is required when TLS is enabled"))
		}
	}

	return issues
}

func validJWKSURL(raw, environment string) bool {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return false
	}
	if strings.EqualFold(parsed.Scheme, "https") {
		return true
	}
	return environment == "local" && strings.EqualFold(parsed.Scheme, "http")
}

func issue(section, field, message string) ValidationIssue {
	return ValidationIssue{
		Section: section,
		Field:   field,
		Message: message,
	}
}
