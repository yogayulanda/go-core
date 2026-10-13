package logger

import (
	"bytes"
	"io"
	"os"
	"strings"

	"go.uber.org/zap/zapcore"
)

type consoleCore struct {
	output zapcore.WriteSyncer
	level  zapcore.LevelEnabler
	tty    bool
	fields []zapcore.Field
}

func newConsoleCore(output zapcore.WriteSyncer, level zapcore.LevelEnabler, tty bool) zapcore.Core {
	return &consoleCore{output: output, level: level, tty: tty}
}

func (c *consoleCore) Enabled(level zapcore.Level) bool {
	return c.level.Enabled(level)
}

func (c *consoleCore) With(fields []zapcore.Field) zapcore.Core {
	cloned := *c
	cloned.fields = append(append([]zapcore.Field(nil), c.fields...), fields...)
	return &cloned
}

func (c *consoleCore) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return checked.AddCore(entry, c)
	}
	return checked
}

func (c *consoleCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	object := zapcore.NewMapObjectEncoder()
	for _, field := range c.fields {
		field.AddTo(object)
	}
	for _, field := range fields {
		field.AddTo(object)
	}
	requestID := consoleField(object.Fields, "labels.request_id", "request_id")
	gtid := consoleField(object.Fields, "labels.gtid", "gtid", "transaction_id")
	userID := consoleField(object.Fields, "user.id", "user_id")
	errSummary := consoleField(object.Fields, "error.message", "err")

	line := formatConsoleLine(entry, requestID, gtid, userID, errSummary, c.tty)
	_, err := c.output.Write([]byte(line))
	return err
}

func (c *consoleCore) Sync() error {
	return c.output.Sync()
}

func isTTY(output io.Writer) bool {
	file, ok := output.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func consoleColor(message string, level zapcore.Level) string {
	switch {
	case strings.HasPrefix(message, "ERROR "):
		return "\x1b[1;31m"
	case strings.HasPrefix(message, "UNKNOWN "):
		return "\x1b[35m"
	case strings.HasPrefix(message, "SLOW SQL "):
		return "\x1b[38;5;208m"
	case strings.HasPrefix(message, "SLOW "):
		return "\x1b[33m"
	case level >= zapcore.ErrorLevel:
		return "\x1b[1;31m"
	case level >= zapcore.WarnLevel:
		return "\x1b[35m"
	case level == zapcore.DebugLevel:
		return "\x1b[90m"
	default:
		return "\x1b[32m"
	}
}

func shortenRequestID(requestID string) string {
	if len(requestID) > 8 {
		return requestID[:8]
	}
	return requestID
}

func formatConsoleLine(entry zapcore.Entry, requestID, gtid, userID, errSummary string, tty bool) string {
	var line bytes.Buffer
	line.WriteString(entry.Time.Format("15:04:05.000"))
	line.WriteByte(' ')
	level := strings.ToUpper(entry.Level.String())
	if tty {
		line.WriteString(consoleColor(entry.Message, entry.Level))
	}
	line.WriteString(level)
	if tty {
		line.WriteString("\x1b[0m")
	}
	line.WriteByte(' ')
	line.WriteString(entry.Message)
	line.WriteString("  req=")
	line.WriteString(shortenRequestID(requestID))
	line.WriteString(" gtid=")
	line.WriteString(gtid)
	line.WriteString(" user=")
	line.WriteString(userID)
	if errSummary != "" {
		line.WriteString(" err=")
		line.WriteString(errSummary)
	}
	line.WriteByte('\n')
	return line.String()
}

func consoleField(fields map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value, ok := fields[key]; ok {
			if text, ok := value.(string); ok {
				return text
			}
		}
	}
	return ""
}
