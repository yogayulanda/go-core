package logger

import (
	"bytes"
	"encoding/json"
	"io"
	"sort"

	"go.uber.org/zap/zapcore"
)

type orderedECSWriter struct {
	output zapcore.WriteSyncer
}

func (w orderedECSWriter) Write(raw []byte) (int, error) {
	trimmed := bytes.TrimSuffix(raw, []byte{'\n'})
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &fields); err != nil {
		return 0, err
	}
	ordered := make([]byte, 0, len(raw))
	ordered = append(ordered, '{')
	wrote := false
	appendField := func(key string) error {
		value, ok := fields[key]
		if !ok {
			return nil
		}
		if wrote {
			ordered = append(ordered, ',')
		}
		encodedKey, err := json.Marshal(key)
		if err != nil {
			return err
		}
		ordered = append(ordered, encodedKey...)
		ordered = append(ordered, ':')
		ordered = append(ordered, value...)
		wrote = true
		delete(fields, key)
		return nil
	}
	for _, key := range []string{"@timestamp", "log.level", "message"} {
		if err := appendField(key); err != nil {
			return 0, err
		}
	}
	remaining := make([]string, 0, len(fields))
	for key := range fields {
		remaining = append(remaining, key)
	}
	sort.Strings(remaining)
	for _, key := range remaining {
		if err := appendField(key); err != nil {
			return 0, err
		}
	}
	ordered = append(ordered, '}', '\n')
	n, err := w.output.Write(ordered)
	if err != nil {
		return 0, err
	}
	if n != len(ordered) {
		return 0, io.ErrShortWrite
	}
	return len(raw), nil
}

func (w orderedECSWriter) Sync() error { return w.output.Sync() }
