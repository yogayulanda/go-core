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
			Environment: "dev",
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
	if !strings.Contains(err.Error(), "InternalJWT Leeway must be >= 0") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_InternalJWTIncludeExcludeConflict(t *testing.T) {
	cfg := &Config{
		App: AppConfig{
			ServiceName: "svc",
			Environment: "dev",
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
			Environment: "dev",
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
			Environment: "dev",
		},
		GRPC: GRPCConfig{
			TLSEnabled: true,
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "GRPC TLSCertFile") || !strings.Contains(err.Error(), "GRPC TLSKeyFile") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_HTTPTLSEnabledWithoutFiles(t *testing.T) {
	cfg := &Config{
		App: AppConfig{
			ServiceName: "svc",
			Environment: "dev",
		},
		HTTP: HTTPConfig{
			TLSEnabled: true,
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "HTTP TLSCertFile") || !strings.Contains(err.Error(), "HTTP TLSKeyFile") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_OTLPInsecureWithCACertConflict(t *testing.T) {
	cfg := &Config{
		App: AppConfig{
			ServiceName: "svc",
			Environment: "dev",
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
	if !strings.Contains(err.Error(), "OTLPCACertFile must be empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_MigrationAutoRunRequiresExplicitDBAndDir(t *testing.T) {
	cfg := &Config{
		App: AppConfig{
			ServiceName: "svc",
			Environment: "dev",
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
			Environment: "dev",
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

func TestValidateIssues_ReturnStructuredIssues(t *testing.T) {
	cfg := &Config{
		Memcached: MemcachedConfig{
			Enabled: true,
		},
	}

	issues := cfg.ValidateIssues()
	if len(issues) == 0 {
		t.Fatalf("expected structured issues")
	}

	foundService := false
	foundMemcached := false
	for _, issue := range issues {
		if issue.Field == "SERVICE_NAME" && issue.Section == "app" {
			foundService = true
		}
		if issue.Field == "MEMCACHED_SERVERS" && issue.Section == "memcached" {
			foundMemcached = true
		}
	}

	if !foundService {
		t.Fatalf("expected SERVICE_NAME issue in app section")
	}
	if !foundMemcached {
		t.Fatalf("expected MEMCACHED_SERVERS issue in memcached section")
	}
}

func TestValidate_ReturnsValidationErrorsType(t *testing.T) {
	cfg := &Config{}

	err := cfg.Validate()
	if err == nil {
		t.Fatalf("expected validation error")
	}

	validationErr, ok := err.(*ValidationErrors)
	if !ok {
		t.Fatalf("expected *ValidationErrors, got %T", err)
	}
	if len(validationErr.Issues) == 0 {
		t.Fatalf("expected validation issues")
	}
}

func TestValidate_AppEnvironmentAllowedValues(t *testing.T) {
	for _, environment := range []string{"local", "dev", "sit", "uat", "prod"} {
		t.Run(environment, func(t *testing.T) {
			cfg := &Config{App: AppConfig{ServiceName: "svc", Environment: environment}}
			if err := cfg.Validate(); err != nil {
				t.Fatalf("valid APP_ENV %q rejected: %v", environment, err)
			}
		})
	}
	for _, environment := range []string{"", "test", "production", "PROD"} {
		t.Run("invalid_"+environment, func(t *testing.T) {
			cfg := &Config{App: AppConfig{ServiceName: "svc", Environment: environment}}
			if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "APP_ENV") {
				t.Fatalf("expected APP_ENV validation error, got %v", err)
			}
		})
	}
}
