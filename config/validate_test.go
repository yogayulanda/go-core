package config

import (
	"strings"
	"testing"
	"time"
)

func TestValidate_InternalJWTNegativeLeeway(t *testing.T) {
	cfg := &Config{
		App: AppConfig{
			ServiceName: "svc",
		},
		Auth: AuthConfig{
			InternalJWT: InternalJWTConfig{
				Enabled:   true,
				PublicKey: "dummy",
				Leeway:    -1 * time.Second,
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "INTERNAL_JWT_LEEWAY must be >= 0") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_InternalJWTIncludeExcludeConflict(t *testing.T) {
	cfg := &Config{
		App: AppConfig{
			ServiceName: "svc",
		},
		Auth: AuthConfig{
			InternalJWT: InternalJWTConfig{
				Enabled:        true,
				PublicKey:      "dummy",
				Leeway:         10 * time.Second,
				IncludeMethods: []string{"/a.b.C/Call"},
				ExcludeMethods: []string{"/grpc.health.v1.Health/Check"},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "cannot be used together") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_InternalJWTIncludeOnlyValid(t *testing.T) {
	cfg := &Config{
		App: AppConfig{
			ServiceName: "svc",
		},
		Auth: AuthConfig{
			InternalJWT: InternalJWTConfig{
				Enabled:        true,
				PublicKey:      "dummy",
				Leeway:         10 * time.Second,
				IncludeMethods: []string{"/a.b.C/Call"},
			},
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_GRPCTLSEnabledWithoutFiles(t *testing.T) {
	cfg := &Config{
		App: AppConfig{
			ServiceName: "svc",
		},
		GRPC: GRPCConfig{
			TLSEnabled: true,
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "GRPC_TLS_CERT_FILE") || !strings.Contains(err.Error(), "GRPC_TLS_KEY_FILE") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_HTTPTLSEnabledWithoutFiles(t *testing.T) {
	cfg := &Config{
		App: AppConfig{
			ServiceName: "svc",
		},
		HTTP: HTTPConfig{
			TLSEnabled: true,
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "HTTP_TLS_CERT_FILE") || !strings.Contains(err.Error(), "HTTP_TLS_KEY_FILE") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_OTLPInsecureWithCACertConflict(t *testing.T) {
	cfg := &Config{
		App: AppConfig{
			ServiceName: "svc",
		},
		Observability: ObservabilityConfig{
			OTLPInsecure:   true,
			OTLPCACertFile: "/tmp/ca.pem",
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "OTEL_EXPORTER_OTLP_CA_CERT_FILE must be empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_MigrationAutoRunRequiresExplicitDBAndDir(t *testing.T) {
	cfg := &Config{
		App: AppConfig{
			ServiceName: "svc",
		},
		Migration: MigrationConfig{
			AutoRun: true,
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "MIGRATION_DB is required") || !strings.Contains(err.Error(), "MIGRATION_DIR is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_MigrationDBMustExistWithNormalizedAlias(t *testing.T) {
	cfg := &Config{
		App: AppConfig{
			ServiceName: "svc",
		},
		Databases: map[string]DBConfig{
			"transaction_history": {
				Driver:          "sqlserver",
				DSN:             "sqlserver://example",
				MaxOpenConns:    10,
				MaxIdleConns:    5,
				ConnMaxIdleTime: time.Minute,
				ConnMaxLifetime: time.Minute,
			},
		},
		Migration: MigrationConfig{
			AutoRun: true,
			DBName:  "Transaction_History",
			Dir:     "migrations/history",
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_DBConnMaxIdleTimeNegative(t *testing.T) {
	cfg := &Config{
		App: AppConfig{
			ServiceName: "svc",
		},
		Databases: map[string]DBConfig{
			"primary": {
				Driver:          "sqlserver",
				DSN:             "sqlserver://example",
				MaxOpenConns:    10,
				MaxIdleConns:    5,
				ConnMaxIdleTime: -1 * time.Second,
				ConnMaxLifetime: time.Minute,
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "DB_PRIMARY_CONN_MAX_IDLE_TIME must be >= 0") {
		t.Fatalf("unexpected error: %v", err)
	}
}
