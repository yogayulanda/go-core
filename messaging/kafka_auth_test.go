package messaging

import (
	"testing"

	"github.com/segmentio/kafka-go/sasl/scram"
	"github.com/yogayulanda/go-core/config"
)

func TestBuildKafkaSecuritySupportsApprovedSASLMechanisms(t *testing.T) {
	tests := []struct {
		name      string
		mechanism string
		wantName  string
	}{
		{name: "plain", mechanism: "PLAIN", wantName: "PLAIN"},
		{name: "scram 256", mechanism: "SCRAM-SHA-256", wantName: "SCRAM-SHA-256"},
		{name: "scram 512", mechanism: "SCRAM-SHA-512", wantName: "SCRAM-SHA-512"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			security, err := buildKafkaSecurity(config.KafkaConfig{Username: "user", Password: "secret", SASLMechanism: test.mechanism})
			if err != nil {
				t.Fatal(err)
			}
			if security.sasl == nil || security.sasl.Name() != test.wantName {
				t.Fatalf("expected mechanism %q, got %#v", test.wantName, security.sasl)
			}
		})
	}
}

func TestBuildKafkaSecurityRejectsIncompleteCredentials(t *testing.T) {
	for _, cfg := range []config.KafkaConfig{
		{Username: "user", SASLMechanism: "PLAIN"},
		{Password: "secret", SASLMechanism: "PLAIN"},
		{Username: "user", Password: "secret", SASLMechanism: "UNKNOWN"},
		{JksPassword: "secret"},
	} {
		if _, err := buildKafkaSecurity(cfg); err == nil {
			t.Errorf("expected invalid Kafka security config to fail: %+v", cfg)
		}
	}
	if _, err := scram.Mechanism(scram.SHA256, "user", "secret"); err != nil {
		t.Fatalf("SCRAM-SHA-256 dependency setup failed: %v", err)
	}
}
