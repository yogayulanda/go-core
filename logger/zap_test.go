package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap/zapcore"
)

func TestResolveLogLocation_DefaultUTC(t *testing.T) {
	loc := resolveLogLocation("")
	if loc.String() != time.UTC.String() {
		t.Fatalf("expected UTC, got %s", loc.String())
	}
}

func TestResolveLogLocation_InvalidFallbackUTC(t *testing.T) {
	loc := resolveLogLocation("Invalid/Timezone")
	if loc.String() != time.UTC.String() {
		t.Fatalf("expected UTC fallback, got %s", loc.String())
	}
}

func TestLoggerNew_ImplementsExpandedInterface(t *testing.T) {
	log, err := New("test-service", "info")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	log.LogService(context.Background(), ServiceLog{Operation: "boot", Status: "success"})
	log.LogDB(context.Background(), DBLog{Operation: "connect", DBName: "primary", Status: "success"})
}

func TestJSONLoggerUsesECSUTCMillisecondsWithoutANSI(t *testing.T) {
	output := &bytes.Buffer{}
	log, err := NewWithOptions("ecs-test", "debug", Options{Environment: "dev", Output: output})
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}
	log.Info(context.Background(), "ecs message", Field{Key: "app.log_type", Value: "service"})

	line := strings.TrimSpace(output.String())
	if !(strings.Index(line, `"@timestamp"`) < strings.Index(line, `"log.level"`) && strings.Index(line, `"log.level"`) < strings.Index(line, `"message"`)) {
		t.Fatalf("ECS field order must start with @timestamp, log.level, message: %s", line)
	}
	if strings.Contains(line, "\x1b[") {
		t.Fatalf("JSON must not contain ANSI codes: %q", line)
	}
	var entry map[string]interface{}
	if err := json.Unmarshal([]byte(line), &entry); err != nil {
		t.Fatalf("decode ECS JSON: %v", err)
	}
	if entry["message"] != "ecs message" || entry["log.level"] != "info" || entry["app.log_type"] != "service" {
		t.Fatalf("unexpected ECS fields: %#v", entry)
	}
	for _, key := range []string{"@timestamp", "log.level", "message", "service.name", "service.environment", "service.version", "ecs.version"} {
		if _, ok := entry[key]; !ok {
			t.Errorf("required ECS key %q missing from %#v", key, entry)
		}
	}
	timestamp, _ := entry["@timestamp"].(string)
	if !strings.HasSuffix(timestamp, "Z") {
		t.Fatalf("timestamp must use UTC millisecond format, got %q", timestamp)
	}
	if _, err := time.Parse("2006-01-02T15:04:05.000Z", timestamp); err != nil {
		t.Fatalf("timestamp must have millisecond precision: %q (%v)", timestamp, err)
	}
}

func TestLocalConsoleEncoderFormatAndEventColors(t *testing.T) {
	output := &bytes.Buffer{}
	log, err := NewWithOptions("local-test", "debug", Options{Environment: "local", Output: output})
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}
	log.Info(context.Background(), "console message", Field{Key: "labels.request_id", Value: "123456789"}, Field{Key: "labels.gtid", Value: "gtid-1"}, Field{Key: "user.id", Value: "user-1"}, Field{Key: "app.secret", Value: "do-not-print"})
	line := output.String()
	for _, expected := range []string{" INFO console message  req=12345678 gtid=gtid-1 user=user-1"} {
		if !strings.Contains(line, expected) {
			t.Fatalf("console line missing %q: %q", expected, line)
		}
	}
	if strings.Contains(line, "do-not-print") || strings.Contains(line, "\x1b[") {
		t.Fatalf("non-TTY console must hide other fields and ANSI: %q", line)
	}
	if got := consoleColor("SLOW SQL 12 SELECT", zapcore.WarnLevel); got != "\x1b[38;5;208m" {
		t.Fatalf("unexpected slow SQL color: %q", got)
	}
	if got := consoleColor("UNKNOWN GET /x", zapcore.WarnLevel); got != "\x1b[35m" {
		t.Fatalf("unexpected unknown color: %q", got)
	}
	if got := consoleColor("ERROR GET /x", zapcore.ErrorLevel); got != "\x1b[1;31m" {
		t.Fatalf("unexpected error color: %q", got)
	}
	if got := consoleColor("ordinary info", zapcore.InfoLevel); got != "\x1b[32m" {
		t.Fatalf("unexpected info color: %q", got)
	}
	if got := consoleColor("debug detail", zapcore.DebugLevel); got != "\x1b[90m" {
		t.Fatalf("unexpected debug color: %q", got)
	}
}

func TestMaskSensitiveValuesAndPreserveSafeValues(t *testing.T) {
	for _, raw := range []string{
		"Bearer eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0.signature",
		"eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0.signature",
		"4111111111111111",
	} {
		got, _ := sanitizeFieldValueWithKeys("payload", raw, nil).(string)
		if got == raw {
			t.Errorf("sensitive value was not masked: %q", raw)
		}
	}
	for _, safe := range []string{"150000", "550e8400-e29b-41d4-a716-446655440000"} {
		got, _ := sanitizeFieldValueWithKeys("payload", safe, nil).(string)
		if got != safe {
			t.Errorf("safe value was modified: %q -> %q", safe, got)
		}
	}
}

func TestAccessLogSlowFlagAndClassMarkers(t *testing.T) {
	output := &bytes.Buffer{}
	log, err := NewWithOptions("access-test", "debug", Options{Environment: "dev", Output: output, Config: Config{SlowRequestThreshold: time.Second}})
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}
	zapLog, ok := log.(*zapLogger)
	if !ok {
		t.Fatalf("expected zapLogger, got %T", log)
	}
	zapLog.LogService(context.Background(), ServiceLog{LogType: "access", Method: "GET", Route: "/fast", StatusCode: 200, ResponseCode: "CORE-0000", Class: "SUCCESS", DurationMs: 999})
	zapLog.LogService(context.Background(), ServiceLog{LogType: "access", Method: "GET", Route: "/slow", StatusCode: 200, ResponseCode: "CORE-0000", Class: "SUCCESS", DurationMs: 1001})
	zapLog.LogService(context.Background(), ServiceLog{LogType: "access", Method: "GET", Route: "/unknown", StatusCode: 500, ResponseCode: "X", Class: "UNKNOWN", DurationMs: 1})
	zapLog.LogService(context.Background(), ServiceLog{LogType: "access", Method: "GET", Route: "/error", StatusCode: 500, ResponseCode: "CORE-9000", Class: "ERROR", DurationMs: 1001})
	zapLog.LogService(context.Background(), ServiceLog{LogType: "access", Method: "GET", Route: "/failed", StatusCode: 422, ResponseCode: "BIZ-1", Class: "FAILED", DurationMs: 1})
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 5 {
		t.Fatalf("expected five access lines, got %d", len(lines))
	}
	var fast, slow, unknown, failed, businessFailed map[string]interface{}
	_ = json.Unmarshal([]byte(lines[0]), &fast)
	_ = json.Unmarshal([]byte(lines[1]), &slow)
	_ = json.Unmarshal([]byte(lines[2]), &unknown)
	_ = json.Unmarshal([]byte(lines[3]), &failed)
	_ = json.Unmarshal([]byte(lines[4]), &businessFailed)
	if _, ok := fast["app.slow"]; ok {
		t.Fatal("fast access log must not contain app.slow")
	}
	if slow["app.slow"] != true || slow["log.level"] != "warn" || !strings.HasPrefix(stringField(slow, "message"), "SLOW ") {
		t.Fatalf("unexpected slow access log: %#v", slow)
	}
	if unknown["log.level"] != "warn" || !strings.HasPrefix(stringField(unknown, "message"), "UNKNOWN ") {
		t.Fatalf("unexpected unknown access log: %#v", unknown)
	}
	if failed["log.level"] != "error" || !strings.HasPrefix(stringField(failed, "message"), "ERROR ") || failed["app.slow"] != true {
		t.Fatalf("unexpected slow error access log: %#v", failed)
	}
	if businessFailed["log.level"] != "info" {
		t.Fatalf("business failure should log at info: %#v", businessFailed)
	}
}

func TestEventLogUsesECSContract(t *testing.T) {
	output := &bytes.Buffer{}
	log, err := NewWithOptions("event-test", "info", Options{Environment: "dev", Output: output})
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}
	log.LogEvent(context.Background(), EventLog{Category: "user", Action: "login", Outcome: "success"})
	var entry map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output.String())), &entry); err != nil {
		t.Fatalf("decode event log: %v", err)
	}
	if entry["event.kind"] != "event" || entry["event.category"] != "user" || entry["event.action"] != "login" || entry["event.outcome"] != "success" || entry["app.log_type"] != "event" {
		t.Fatalf("unexpected ECS event fields: %#v", entry)
	}
	if entry["message"] != "event login success" {
		t.Fatalf("unexpected event message: %v", entry["message"])
	}
}

func TestDBLogSlowMarkerAndThreshold(t *testing.T) {
	output := &bytes.Buffer{}
	log, err := NewWithOptions("db-slow-test", "debug", Options{Environment: "dev", Output: output, Config: Config{SlowQueryThreshold: time.Second}})
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}
	log.LogDB(context.Background(), DBLog{Operation: "query", Query: "SELECT 1", Status: "success", DurationMs: 1001})
	var entry map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output.String())), &entry); err != nil {
		t.Fatalf("decode log: %v", err)
	}
	if entry["log.level"] != "warn" || entry["app.log_type"] != "db" || entry["app.slow"] != true || !strings.HasPrefix(stringField(entry, "message"), "SLOW SQL 1001 ") {
		t.Fatalf("unexpected slow DB log: %#v", entry)
	}
}

func stringField(fields map[string]interface{}, key string) string {
	value, _ := fields[key].(string)
	return value
}

func TestDBLogBelowThresholdOmitsSlowFlag(t *testing.T) {
	output := &bytes.Buffer{}
	log, err := NewWithOptions("db-fast-test", "debug", Options{Environment: "dev", Output: output, Config: Config{SlowQueryThreshold: time.Second}})
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}
	log.LogDB(context.Background(), DBLog{Operation: "query", Query: "SELECT 1", Status: "success", DurationMs: 999})
	var entry map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output.String())), &entry); err != nil {
		t.Fatalf("decode log: %v", err)
	}
	if _, ok := entry["app.slow"]; ok {
		t.Fatalf("normal DB log must omit app.slow: %#v", entry)
	}
}
