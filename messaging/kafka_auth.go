package messaging

import (
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"
	"github.com/yogayulanda/go-core/config"
	"github.com/yogayulanda/go-core/messaging/internal/kafkatls"
)

type kafkaSecurity struct {
	tls  *tls.Config
	sasl sasl.Mechanism
}

func buildKafkaSecurity(cfg config.KafkaConfig) (kafkaSecurity, error) {
	security := kafkaSecurity{}
	username := strings.TrimSpace(cfg.Username)
	mechanism := strings.ToUpper(strings.TrimSpace(cfg.SASLMechanism))
	if username != "" {
		if cfg.Password == "" {
			return kafkaSecurity{}, fmt.Errorf("KAFKA_PASSWORD is required when KAFKA_USERNAME is set")
		}
		switch mechanism {
		case "PLAIN":
			security.sasl = plain.Mechanism{Username: username, Password: cfg.Password}
		case "SCRAM-SHA-256":
			configured, err := scram.Mechanism(scram.SHA256, username, cfg.Password)
			if err != nil {
				return kafkaSecurity{}, fmt.Errorf("configure Kafka SCRAM-SHA-256: %w", err)
			}
			security.sasl = configured
		case "SCRAM-SHA-512":
			configured, err := scram.Mechanism(scram.SHA512, username, cfg.Password)
			if err != nil {
				return kafkaSecurity{}, fmt.Errorf("configure Kafka SCRAM-SHA-512: %w", err)
			}
			security.sasl = configured
		default:
			return kafkaSecurity{}, fmt.Errorf("unsupported KAFKA_SASL_MECHANISM %q", mechanism)
		}
	} else if cfg.Password != "" || mechanism != "" {
		return kafkaSecurity{}, fmt.Errorf("KAFKA_USERNAME is required when SASL credentials or mechanism are set")
	}

	if strings.TrimSpace(cfg.JksFile) != "" {
		tlsConfig, err := kafkatls.Load(cfg.JksFile, cfg.JksPassword)
		if err != nil {
			return kafkaSecurity{}, fmt.Errorf("load Kafka truststore: %w", err)
		}
		security.tls = tlsConfig
	} else if cfg.JksPassword != "" {
		return kafkaSecurity{}, fmt.Errorf("KAFKA_JKS_FILE is required when KAFKA_JKS_PASSWORD is set")
	}
	return security, nil
}

func buildKafkaDialer(cfg config.KafkaConfig) (*kafka.Dialer, error) {
	security, err := buildKafkaSecurity(cfg)
	if err != nil {
		return nil, err
	}
	return &kafka.Dialer{ClientID: cfg.ClientID, Timeout: 5 * time.Second, TLS: security.tls, SASLMechanism: security.sasl}, nil
}

func buildKafkaTransport(cfg config.KafkaConfig) (*kafka.Transport, error) {
	security, err := buildKafkaSecurity(cfg)
	if err != nil {
		return nil, err
	}
	return &kafka.Transport{ClientID: cfg.ClientID, DialTimeout: 5 * time.Second, IdleTimeout: 30 * time.Second, TLS: security.tls, SASL: security.sasl}, nil
}
