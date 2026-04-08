package config

import (
	"errors"
	"strings"
)

func (c *Config) Validate() error {

	var errs []string

	if c.App.ServiceName == "" {
		errs = append(errs, "SERVICE_NAME is required")
	}

	for name, db := range c.Databases {
		if db.Driver == "" {
			errs = append(errs, "DB_"+strings.ToUpper(name)+"_DRIVER is required")
		}
		if db.DSN == "" {
			errs = append(errs, "DB_"+strings.ToUpper(name)+"_DSN or DB_"+strings.ToUpper(name)+"_{HOST,PORT,NAME,USER} is required")
		}
		if db.MaxOpenConns <= 0 {
			errs = append(errs, "DB_"+strings.ToUpper(name)+"_MAX_OPEN_CONNS must be > 0")
		}
		if db.MaxIdleConns < 0 {
			errs = append(errs, "DB_"+strings.ToUpper(name)+"_MAX_IDLE_CONNS must be >= 0")
		}
		if db.ConnMaxIdleTime < 0 {
			errs = append(errs, "DB_"+strings.ToUpper(name)+"_CONN_MAX_IDLE_TIME must be >= 0")
		}
		if db.Port < 0 {
			errs = append(errs, "DB_"+strings.ToUpper(name)+"_PORT must be >= 0")
		}
	}

	if c.Migration.AutoRun {
		migrationDB := NormalizeDBAlias(c.Migration.DBName)
		if migrationDB == "" {
			errs = append(errs, "MIGRATION_DB is required when MIGRATION_AUTO_RUN=true")
		} else if !hasDatabaseAlias(c.Databases, migrationDB) {
			errs = append(errs, "MIGRATION_DB must exist in DB_LIST when MIGRATION_AUTO_RUN=true")
		}

		if strings.TrimSpace(c.Migration.Dir) == "" {
			errs = append(errs, "MIGRATION_DIR is required when MIGRATION_AUTO_RUN=true")
		}
		if c.Migration.LockEnabled && c.Migration.LockTimeout <= 0 {
			errs = append(errs, "MIGRATION_LOCK_TIMEOUT must be > 0 when MIGRATION_LOCK_ENABLED=true")
		}
	}

	if c.Kafka.Enabled && len(c.Kafka.Brokers) == 0 {
		errs = append(errs, "KAFKA_BROKERS required when KAFKA_ENABLED=true")
	}

	if c.Observability.OTLPInsecure && strings.TrimSpace(c.Observability.OTLPCACertFile) != "" {
		errs = append(errs, "OTEL_EXPORTER_OTLP_CA_CERT_FILE must be empty when OTEL_EXPORTER_OTLP_INSECURE=true")
	}

	if c.Redis.Enabled && c.Redis.Address == "" {
		errs = append(errs, "REDIS_ADDRESS required when REDIS_ENABLED=true")
	}

	if c.Memcached.Enabled {
		if len(c.Memcached.Servers) == 0 {
			errs = append(errs, "MEMCACHED_SERVERS (or MEMCACHED_ADDRESS) required when MEMCACHED_ENABLED=true")
		}
		if c.Memcached.Timeout <= 0 {
			errs = append(errs, "MEMCACHED_TIMEOUT must be > 0 when MEMCACHED_ENABLED=true")
		}
	}

	if c.Auth.InternalJWT.Enabled && strings.TrimSpace(c.Auth.InternalJWT.PublicKey) == "" {
		errs = append(errs, "INTERNAL_JWT_PUBLIC_KEY required when INTERNAL_JWT_ENABLED=true")
	}
	if c.Auth.InternalJWT.Enabled && c.Auth.InternalJWT.Leeway < 0 {
		errs = append(errs, "INTERNAL_JWT_LEEWAY must be >= 0 when INTERNAL_JWT_ENABLED=true")
	}
	if c.Auth.InternalJWT.Enabled &&
		len(c.Auth.InternalJWT.IncludeMethods) > 0 &&
		len(c.Auth.InternalJWT.ExcludeMethods) > 0 {
		errs = append(errs, "INTERNAL_JWT_INCLUDE_METHODS and INTERNAL_JWT_EXCLUDE_METHODS cannot be used together")
	}

	if c.GRPC.TLSEnabled {
		if strings.TrimSpace(c.GRPC.TLSCertFile) == "" {
			errs = append(errs, "GRPC_TLS_CERT_FILE required when GRPC_TLS_ENABLED=true")
		}
		if strings.TrimSpace(c.GRPC.TLSKeyFile) == "" {
			errs = append(errs, "GRPC_TLS_KEY_FILE required when GRPC_TLS_ENABLED=true")
		}
	}

	if c.HTTP.TLSEnabled {
		if strings.TrimSpace(c.HTTP.TLSCertFile) == "" {
			errs = append(errs, "HTTP_TLS_CERT_FILE required when HTTP_TLS_ENABLED=true")
		}
		if strings.TrimSpace(c.HTTP.TLSKeyFile) == "" {
			errs = append(errs, "HTTP_TLS_KEY_FILE required when HTTP_TLS_ENABLED=true")
		}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}

	return nil
}
