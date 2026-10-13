package messaging

import (
	"github.com/segmentio/kafka-go"
	"github.com/yogayulanda/go-core/config"
)

func NewKafkaDialer(cfg config.KafkaConfig) (*kafka.Dialer, error) {
	return buildKafkaDialer(cfg)
}
