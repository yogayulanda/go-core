package config

import (
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

	for name, db := range c.Databases {
		if db.Driver == "" {
			issues = append(issues, issue("database", "DB_"+strings.ToUpper(name)+"_DRIVER", "DB_"+strings.ToUpper(name)+"_DRIVER is required"))
		}
		if db.DSN == "" {
			issues = append(issues, issue("database", "DB_"+strings.ToUpper(name)+"_DSN", "DB_"+strings.ToUpper(name)+"_DSN or DB_"+strings.ToUpper(name)+"_{HOST,PORT,NAME,USER} is required"))
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
		if db.Port < 0 {
			issues = append(issues, issue("database", "DB_"+strings.ToUpper(name)+"_PORT", "DB_"+strings.ToUpper(name)+"_PORT must be >= 0"))
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
		issues = append(issues, issue("kafka", "KAFKA_BROKERS", "KAFKA_BROKERS required when KAFKA_ENABLED=true"))
	}

	if c.Observability.OTLPInsecure && strings.TrimSpace(c.Observability.OTLPCACertFile) != "" {
		issues = append(issues, issue("observability", "OTEL_EXPORTER_OTLP_CA_CERT_FILE", "OTEL_EXPORTER_OTLP_CA_CERT_FILE must be empty when OTEL_EXPORTER_OTLP_INSECURE=true"))
	}

	if c.Redis.Enabled && c.Redis.Address == "" {
		issues = append(issues, issue("redis", "REDIS_ADDRESS", "REDIS_ADDRESS required when REDIS_ENABLED=true"))
	}

	if c.Memcached.Enabled {
		if len(c.Memcached.Servers) == 0 {
			issues = append(issues, issue("memcached", "MEMCACHED_SERVERS", "MEMCACHED_SERVERS (or MEMCACHED_ADDRESS) required when MEMCACHED_ENABLED=true"))
		}
		if c.Memcached.Timeout <= 0 {
			issues = append(issues, issue("memcached", "MEMCACHED_TIMEOUT", "MEMCACHED_TIMEOUT must be > 0 when MEMCACHED_ENABLED=true"))
		}
	}

	if c.Auth.InternalJWT.Enabled && strings.TrimSpace(c.Auth.InternalJWT.PublicKey) == "" {
		issues = append(issues, issue("auth", "INTERNAL_JWT_PUBLIC_KEY", "INTERNAL_JWT_PUBLIC_KEY required when INTERNAL_JWT_ENABLED=true"))
	}
	if c.Auth.InternalJWT.Enabled && c.Auth.InternalJWT.Leeway < 0 {
		issues = append(issues, issue("auth", "INTERNAL_JWT_LEEWAY", "INTERNAL_JWT_LEEWAY must be >= 0 when INTERNAL_JWT_ENABLED=true"))
	}
	if c.Auth.InternalJWT.Enabled &&
		len(c.Auth.InternalJWT.IncludeMethods) > 0 &&
		len(c.Auth.InternalJWT.ExcludeMethods) > 0 {
		issues = append(issues, issue("auth", "INTERNAL_JWT_INCLUDE_METHODS", "INTERNAL_JWT_INCLUDE_METHODS and INTERNAL_JWT_EXCLUDE_METHODS cannot be used together"))
	}

	if c.GRPC.TLSEnabled {
		if strings.TrimSpace(c.GRPC.TLSCertFile) == "" {
			issues = append(issues, issue("grpc", "GRPC_TLS_CERT_FILE", "GRPC_TLS_CERT_FILE required when GRPC_TLS_ENABLED=true"))
		}
		if strings.TrimSpace(c.GRPC.TLSKeyFile) == "" {
			issues = append(issues, issue("grpc", "GRPC_TLS_KEY_FILE", "GRPC_TLS_KEY_FILE required when GRPC_TLS_ENABLED=true"))
		}
	}

	if c.HTTP.TLSEnabled {
		if strings.TrimSpace(c.HTTP.TLSCertFile) == "" {
			issues = append(issues, issue("http", "HTTP_TLS_CERT_FILE", "HTTP_TLS_CERT_FILE required when HTTP_TLS_ENABLED=true"))
		}
		if strings.TrimSpace(c.HTTP.TLSKeyFile) == "" {
			issues = append(issues, issue("http", "HTTP_TLS_KEY_FILE", "HTTP_TLS_KEY_FILE required when HTTP_TLS_ENABLED=true"))
		}
	}

	return issues
}

func issue(section, field, message string) ValidationIssue {
	return ValidationIssue{
		Section: section,
		Field:   field,
		Message: message,
	}
}
