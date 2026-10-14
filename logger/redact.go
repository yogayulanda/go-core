package logger

import (
	"fmt"
	"regexp"
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
	"session",
	"phone",
	"msisdn",
	"account",
	"cif",
	"nik",
}

var cardNumberPattern = regexp.MustCompile(`(?:\d[ -]?){13,19}`)

func sanitizeFieldValueWithKeys(key string, value interface{}, maskKeys []string) interface{} {
	if isSensitiveKeyWithKeys(key, maskKeys) {
		return maskSensitiveValue(value)
	}
	switch v := value.(type) {
	case string:
		return sanitizeSensitiveValue(v)
	case map[string]interface{}:
		out := make(map[string]interface{}, len(v))
		for nestedKey, nestedValue := range v {
			out[nestedKey] = sanitizeFieldValueWithKeys(nestedKey, nestedValue, maskKeys)
		}
		return out
	case map[string]string:
		out := make(map[string]string, len(v))
		for nestedKey, nestedValue := range v {
			if isSensitiveKeyWithKeys(nestedKey, maskKeys) {
				out[nestedKey] = maskStringKeepLastN(nestedValue, 2)
			} else {
				out[nestedKey] = sanitizeSensitiveValue(nestedValue)
			}
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(v))
		for i, item := range v {
			out[i] = sanitizeFieldValueWithKeys(key, item, maskKeys)
		}
		return out
	default:
		return value
	}
}

func sanitizeFieldValue(key string, value interface{}) interface{} {
	return sanitizeFieldValueWithKeys(key, value, nil)
}

func isSensitiveKeyWithKeys(key string, extra []string) bool {
	k := strings.ToLower(strings.TrimSpace(key))
	if k == "" {
		return false
	}
	for _, hint := range sensitiveKeyHints {
		if strings.Contains(k, hint) {
			return true
		}
	}
	for _, hint := range extra {
		hint = strings.ToLower(strings.TrimSpace(hint))
		if hint != "" && strings.Contains(k, hint) {
			return true
		}
	}
	return false
}

func sanitizeSensitiveValue(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if len(trimmed) > 7 && strings.EqualFold(trimmed[:7], "Bearer ") {
		return "Bearer " + maskStringKeepLastN(strings.TrimSpace(trimmed[7:]), 4)
	}
	parts := strings.Split(trimmed, ".")
	if len(parts) == 3 && isJWTPart(parts[0]) && isJWTPart(parts[1]) && isJWTPart(parts[2]) {
		return maskStringKeepLastN(trimmed, 4)
	}
	match := cardNumberPattern.FindString(trimmed)
	if match == "" {
		return raw
	}
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, match)
	if len(digits) >= 13 && len(digits) <= 19 && validLuhn(digits) {
		return maskStringKeepLastN(digits, 4)
	}
	return raw
}

func isJWTPart(part string) bool {
	if part == "" {
		return false
	}
	for _, r := range part {
		if !(r >= 'a' && r <= 'z') && !(r >= 'A' && r <= 'Z') && !(r >= '0' && r <= '9') && r != '-' && r != '_' {
			return false
		}
	}
	return true
}

func validLuhn(number string) bool {
	sum := 0
	double := false
	for i := len(number) - 1; i >= 0; i-- {
		digit := int(number[i] - '0')
		if double {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
		double = !double
	}
	return sum%10 == 0
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
