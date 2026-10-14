package logger

import (
	"context"
	"io"
	"strings"

	"google.golang.org/grpc/grpclog"
)

type grpcLogWriter struct {
	log   Logger
	level string
}

func (w grpcLogWriter) Write(raw []byte) (int, error) {
	message := strings.TrimSpace(string(raw))
	if message == "" || w.log == nil || isRoutineGRPCLog(message) {
		return len(raw), nil
	}
	if w.level == "error" {
		w.log.Error(context.Background(), message)
	} else {
		w.log.Warn(context.Background(), message)
	}
	return len(raw), nil
}

func NewGRPCLoggerV2(log Logger) grpclog.LoggerV2 {
	return grpclog.NewLoggerV2(io.Discard, grpcLogWriter{log: log, level: "warn"}, grpcLogWriter{log: log, level: "error"})
}

func isRoutineGRPCLog(message string) bool {
	message = strings.ToLower(message)
	for _, routine := range []string{"transport is closing", "client connection is closing", "clientconn closing", "eof"} {
		if strings.Contains(message, routine) {
			return true
		}
	}
	return false
}

var _ io.Writer = grpcLogWriter{}
