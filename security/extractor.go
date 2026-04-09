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

func ExtractMetadataSummary(ctx context.Context) map[string]interface{} {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return map[string]interface{}{
			"metadata_present": false,
			"claim_count":      0,
		}
	}

	claimCount := 0
	for key, values := range md {
		k := strings.ToLower(strings.TrimSpace(key))
		if !strings.HasPrefix(k, "x-claim-") || len(values) == 0 {
			continue
		}
		claimCount++
	}

	return map[string]interface{}{
		"metadata_present":  true,
		"subject_present":   strings.TrimSpace(firstMetadataValue(md, "x-subject")) != "",
		"session_present":   strings.TrimSpace(firstMetadataValue(md, "x-session-id")) != "",
		"role_present":      strings.TrimSpace(firstMetadataValue(md, "x-role")) != "",
		"claim_count":       claimCount,
		"attribute_present": claimCount > 0,
	}
}

func firstMetadataValue(md metadata.MD, key string) string {
	values := md.Get(key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func addIfNotEmpty(attrs map[string]string, key, value string) {
	v := strings.TrimSpace(value)
	if v == "" {
		return
	}
	attrs[key] = v
}
