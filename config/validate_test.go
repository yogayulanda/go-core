package config

import (
	"strings"
	"testing"
	"time"
)

func TestValidate_InternalJWTNegativeLeeway(t *testing.T) {
	cfg := &Config{
		App: AppConfig{
			ServiceName:     "svc",
			Environment:     "dev",
			ShutdownTimeout: 25 * time.Second,
		},
		Auth: AuthConfig{
			InternalJWT: InternalJWTConfig{
				Enabled:   true,
				PublicKey: "dummy",
				Leeway:    -1 * time.Second,
			},
		},
		HTTP: HTTPConfig{
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       15 * time.Second,
			WriteTimeout:      60 * time.Second,
			IdleTimeout:       120 * time.Second,
			MaxBodyBytes:      1 << 20,
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

func TestValidate_KafkaSASLAndJKSRequirements(t *testing.T) {
	base := Config{App: AppConfig{ServiceName: "svc", Environment: "dev", ShutdownTimeout: 25 * time.Second}}
	tests := []struct {
		name   string
		kafka  KafkaConfig
		failed string
	}{
		{name: "username requires password", kafka: KafkaConfig{Username: "user", SASLMechanism: "PLAIN"}, failed: "KAFKA_PASSWORD"},
		{name: "username requires supported mechanism", kafka: KafkaConfig{Username: "user", Password: "secret", SASLMechanism: "UNKNOWN"}, failed: "KAFKA_SASL_MECHANISM"},
		{name: "mechanism requires username", kafka: KafkaConfig{Password: "secret", SASLMechanism: "PLAIN"}, failed: "KAFKA_USERNAME"},
		{name: "JKS password requires file", kafka: KafkaConfig{JksPassword: "secret"}, failed: "KAFKA_JKS_FILE"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := base
			cfg.Kafka = test.kafka
			err := cfg.Validate()
			if err == nil || !strings.Contains(err.Error(), test.failed) {
				t.Fatalf("expected %s validation error, got %v", test.failed, err)
			}
		})
	}
}

func TestValidate_InternalJWTIncludeExcludeConflict(t *testing.T) {
	cfg := &Config{
		App: AppConfig{
			ServiceName:     "svc",
			Environment:     "dev",
			ShutdownTimeout: 25 * time.Second,
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
	cfg := validBaseConfig()
	cfg.Auth = AuthConfig{
		InternalJWT: InternalJWTConfig{
			Enabled:        true,
			PublicKey:      "dummy",
			Leeway:         10 * time.Second,
			IncludeMethods: []string{"/a.b.C/Call"},
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_InternalJWTKeySourceOptions(t *testing.T) {
	tests := []struct {
		name        string
		publicKey   string
		jwksURL     string
		environment string
		wantError   string
	}{
		{name: "public key only", publicKey: "public-key", environment: "prod"},
		{name: "jwks only", jwksURL: "https://keys.example.test/jwks", environment: "prod"},
		{name: "both sources", publicKey: "public-key", jwksURL: "https://keys.example.test/jwks", environment: "prod", wantError: "InternalJWT requires exactly one of PublicKey or JWKSEndpoint; both are set; remove one"},
		{name: "no source", environment: "prod", wantError: "InternalJWT requires exactly one of PublicKey or JWKSEndpoint"},
		{name: "http jwks outside local", jwksURL: "http://keys.example.test/jwks", environment: "prod", wantError: "must be an https URL"},
		{name: "http jwks in local", jwksURL: "http://keys.example.test/jwks", environment: "local"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := validBaseConfig()
			cfg.App.Environment = test.environment
			cfg.Auth.InternalJWT = InternalJWTConfig{Enabled: true, PublicKey: test.publicKey, JWKSEndpoint: test.jwksURL}
			err := cfg.Validate()
			if test.wantError == "" {
				if err != nil {
					t.Fatalf("unexpected validation error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("expected validation error %q, got %v", test.wantError, err)
			}
		})
	}
}

func TestValidate_GRPCTLSEnabledWithoutFiles(t *testing.T) {
	cfg := &Config{
		App: AppConfig{
			ServiceName:     "svc",
			Environment:     "dev",
			ShutdownTimeout: 25 * time.Second,
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
	cfg := validBaseConfig()
	cfg.Databases = map[string]DBConfig{
		"transaction_history": {
			Driver:          "sqlserver",
			DSN:             "sqlserver://example",
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxIdleTime: time.Minute,
			ConnMaxLifetime: time.Minute,
		},
	}
	cfg.Migration = MigrationConfig{AutoRun: true, DBName: "Transaction_History", Dir: "migrations/history"}

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
		HTTP: HTTPConfig{
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       15 * time.Second,
			WriteTimeout:      60 * time.Second,
			IdleTimeout:       120 * time.Second,
			MaxBodyBytes:      1 << 20,
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
			cfg := validBaseConfig()
			cfg.App.Environment = environment
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

func validBaseConfig() *Config {
	return &Config{
		App:  AppConfig{ServiceName: "svc", Environment: "dev", ShutdownTimeout: 25 * time.Second},
		GRPC: GRPCConfig{Port: 50051},
		HTTP: HTTPConfig{Port: 8080, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 120 * time.Second, MaxBodyBytes: 1 << 20},
	}
}

func TestValidate_PprofForbiddenInProduction(t *testing.T) {
	cfg := &Config{
		App:  AppConfig{ServiceName: "svc", Environment: "prod", ShutdownTimeout: 25 * time.Second},
		HTTP: HTTPConfig{PprofEnabled: true, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 120 * time.Second, MaxBodyBytes: 1 << 20},
	}
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "pprof is forbidden") {
		t.Fatalf("expected production pprof validation error, got %v", err)
	}
}

func TestValidate_HTTPTimeoutRelationship(t *testing.T) {
	cfg := &Config{
		App:  AppConfig{ServiceName: "svc", Environment: "dev", ShutdownTimeout: 25 * time.Second},
		HTTP: HTTPConfig{ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 120 * time.Second, MaxBodyBytes: 1 << 20},
	}
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "WriteTimeout must be greater") {
		t.Fatalf("expected timeout relationship validation error, got %v", err)
	}
}

func TestValidate_PortRangesAndDistinctListeners(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*Config)
		want  string
	}{
		{
			name: "grpc port zero",
			setup: func(cfg *Config) {
				cfg.GRPC.Port = 0
			},
			want: "GRPC_PORT must be between 1 and 65535",
		},
		{
			name: "http port too high",
			setup: func(cfg *Config) {
				cfg.HTTP.Port = 70000
			},
			want: "HTTP_PORT must be between 1 and 65535",
		},
		{
			name: "listener ports equal",
			setup: func(cfg *Config) {
				cfg.HTTP.Port = cfg.GRPC.Port
			},
			want: "GRPC_PORT and HTTP_PORT must be different",
		},
		{
			name: "database port zero",
			setup: func(cfg *Config) {
				cfg.Databases = map[string]DBConfig{"primary": {}}
			},
			want: "DB_PRIMARY_PORT must be between 1 and 65535",
		},
		{
			name: "database port too high",
			setup: func(cfg *Config) {
				cfg.Databases = map[string]DBConfig{"primary": {DSN: "mysql://example", Port: 70000}}
			},
			want: "DB_PRIMARY_PORT must be between 1 and 65535",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := validBaseConfig()
			test.setup(cfg)
			if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q validation error, got %v", test.want, err)
			}
		})
	}
}
