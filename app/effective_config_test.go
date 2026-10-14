package app

import (
	"testing"
	"time"

	"github.com/yogayulanda/go-core/config"
)

func TestEffectiveConfigFieldsIncludeRuntimeValuesAndRedactSecrets(t *testing.T) {
	cfg := &config.Config{
		App:       config.AppConfig{ServiceName: "effective-test", Environment: "dev", ShutdownTimeout: 25 * time.Second},
		HTTP:      config.HTTPConfig{WriteTimeout: 60 * time.Second},
		GRPC:      config.GRPCConfig{TLSKeyFile: "/etc/service/grpc-private-key.pem"},
		Databases: map[string]config.DBConfig{"primary": {DSN: "user:password@tcp(db)", Password: "db-secret"}},
		Auth:      config.AuthConfig{Signature: config.SignatureConfig{MasterKey: "signature-secret"}},
		Kafka:     config.KafkaConfig{JksPassword: "jks-secret"},
	}
	fields := effectiveConfigFields(cfg)
	values := make(map[string]interface{}, len(fields))
	for _, field := range fields {
		values[field.Key] = field.Value
	}
	if values["http.write_timeout"] != "1m0s" {
		t.Fatalf("effective config missing formatted HTTP timeout: %#v", values["http.write_timeout"])
	}
	for _, key := range []string{"databases.primary.dsn", "databases.primary.password", "auth.signature.master_key", "kafka.jks_password", "grpc.tls_key_file"} {
		if values[key] != "<REDACTED>" {
			t.Errorf("expected %s to be redacted, got %#v", key, values[key])
		}
	}
	for _, raw := range []string{"user:password@tcp(db)", "db-secret", "signature-secret", "jks-secret", "/etc/service/grpc-private-key.pem"} {
		for key, value := range values {
			if value == raw {
				t.Errorf("effective config field %s leaked secret %q", key, raw)
			}
		}
	}
}
