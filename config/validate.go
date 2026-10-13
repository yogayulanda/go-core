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
	if c.App.Environment == "" {
		issues = append(issues, issue("app", "APP_ENV", "APP_ENV is required"))
	} else {
		switch c.App.Environment {
		case "local", "dev", "sit", "uat", "prod":
		default:
			issues = append(issues, issue("app", "APP_ENV", "APP_ENV must be one of local, dev, sit, uat, prod"))
		}
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
		issues = append(issues, issue("kafka", "KAFKA_BROKERS", "KAFKA_BROKERS must not be empty when Kafka is enabled"))
	}
	if c.Kafka.Username != "" {
		switch c.Kafka.SASLMechanism {
		case "PLAIN", "SCRAM-SHA-256", "SCRAM-SHA-512":
		default:
			issues = append(issues, issue("kafka", "KAFKA_SASL_MECHANISM", "KAFKA_SASL_MECHANISM must be PLAIN, SCRAM-SHA-256, or SCRAM-SHA-512 when KAFKA_USERNAME is set"))
		}
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

	if c.Auth.InternalJWT.Enabled && strings.TrimSpace(c.Auth.InternalJWT.PublicKey) == "" {
		issues = append(issues, issue("auth", "InternalJWT.PublicKey", "InternalJWT PublicKey is required when enabled"))
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

func issue(section, field, message string) ValidationIssue {
	return ValidationIssue{
		Section: section,
		Field:   field,
		Message: message,
	}
}
