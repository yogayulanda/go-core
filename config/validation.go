package config

import "strings"

func joinValidationMessages(messages []string) string {
	return strings.Join(messages, "; ")
}
