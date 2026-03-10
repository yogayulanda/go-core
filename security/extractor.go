package security

import (
	"context"
	"strings"

	"google.golang.org/grpc/metadata"
)

func ExtractFromMetadata(ctx context.Context) *Claims {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil
	}

	get := func(key string) string {
		values := md.Get(key)
		if len(values) > 0 {
			return values[0]
		}
		return ""
	}

	subject := strings.TrimSpace(get("x-subject"))
	sessionID := get("x-session-id")
	role := strings.TrimSpace(get("x-role"))

	attributes := map[string]string{}
	for key, values := range md {
		k := strings.ToLower(strings.TrimSpace(key))
		if !strings.HasPrefix(k, "x-claim-") || len(values) == 0 {
			continue
		}
		attrName := strings.TrimSpace(strings.TrimPrefix(k, "x-claim-"))
		if attrName == "" {
			continue
		}
		attributes[attrName] = values[0]
	}

	if subject == "" && sessionID == "" && role == "" && len(attributes) == 0 {
		return nil
	}

	return &Claims{
		Subject:    subject,
		SessionID:  sessionID,
		Role:       role,
		Attributes: attributes,
	}
}

func addIfNotEmpty(attrs map[string]string, key, value string) {
	v := strings.TrimSpace(value)
	if v == "" {
		return
	}
	attrs[key] = v
}
