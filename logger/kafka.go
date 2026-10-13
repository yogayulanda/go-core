package logger

import (
	"context"
	"io"
	"log"
	"strings"
)

type kafkaErrorWriter struct {
	logger Logger
}

func (w kafkaErrorWriter) Write(raw []byte) (int, error) {
	message := strings.TrimSpace(string(raw))
	if message == "" || w.logger == nil {
		return len(raw), nil
	}
	w.logger.Error(context.Background(), "kafka-go error", Field{Key: "error.message", Value: message})
	return len(raw), nil
}

func NewKafkaErrorLogger(logWriter Logger) *log.Logger {
	return log.New(kafkaErrorWriter{logger: logWriter}, "", 0)
}

var _ io.Writer = kafkaErrorWriter{}
