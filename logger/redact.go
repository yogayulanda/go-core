package logger

import (
	"fmt"
	"strings"
)

var sensitiveKeyHints = []string{
	"password",
	"passwd",
	"secret",
	"token",
	"authorization",
	"apikey",
	"api_key",
	"pin",
	"otp",
	"cvv",
	"card",
	"private_key",
}

func sanitizeFieldValue(key string, value interface{}) interface{} {
	if isSensitiveKey(key) {
		return maskSensitiveValue(value)
	}

	switch v := value.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(v))
		for mk, mv := range v {
			out[mk] = sanitizeFieldValue(mk, mv)
		}
		return out
	case map[string]string:
		out := make(map[string]string, len(v))
		for mk, mv := range v {
			if isSensitiveKey(mk) {
				out[mk] = maskStringKeepLastN(mv, 2)
				continue
			}
			out[mk] = mv
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(v))
		for i, item := range v {
			out[i] = sanitizeFieldValue(key, item)
		}
		return out
	default:
		return value
	}
}

func isSensitiveKey(key string) bool {
	k := strings.ToLower(strings.TrimSpace(key))
	if k == "" {
		return false
	}
	for _, hint := range sensitiveKeyHints {
		if strings.Contains(k, hint) {
			return true
		}
	}
	return false
}

func maskSensitiveValue(v interface{}) string {
	if v == nil {
		return "**"
	}
	return maskStringKeepLastN(fmt.Sprintf("%v", v), 2)
}

func maskStringKeepLastN(raw string, keep int) string {
	if keep < 0 {
		keep = 0
	}
	s := strings.TrimSpace(raw)
	if s == "" {
		return "**"
	}

	runes := []rune(s)
	if len(runes) <= keep {
		return strings.Repeat("*", len(runes))
	}

	maskLen := len(runes) - keep
	return strings.Repeat("*", maskLen) + string(runes[maskLen:])
}
