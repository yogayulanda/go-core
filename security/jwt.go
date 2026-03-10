package security

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/yogayulanda/go-core/config"
)

type InternalJWTVerifier struct {
	enabled        bool
	publicKey      *rsa.PublicKey
	issuer         string
	audience       string
	leeway         time.Duration
	includeMethods map[string]struct{}
	excludeMethods map[string]struct{}
}

func NewInternalJWTVerifier(cfg config.InternalJWTConfig) (*InternalJWTVerifier, error) {
	verifier := &InternalJWTVerifier{
		enabled:        cfg.Enabled,
		issuer:         strings.TrimSpace(cfg.Issuer),
		audience:       strings.TrimSpace(cfg.Audience),
		leeway:         normalizeLeeway(cfg.Leeway),
		includeMethods: normalizeMethodSet(cfg.IncludeMethods),
		excludeMethods: normalizeMethodSet(cfg.ExcludeMethods),
	}

	if !cfg.Enabled {
		return verifier, nil
	}

	key, err := parseRSAPublicKey(cfg.PublicKey)
	if err != nil {
		return nil, err
	}
	verifier.publicKey = key

	return verifier, nil
}

func (v *InternalJWTVerifier) Enabled() bool {
	return v != nil && v.enabled
}

func (v *InternalJWTVerifier) Verify(token string) (*Claims, error) {
	if !v.Enabled() {
		return nil, nil
	}
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("authorization token is empty")
	}

	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{"RS256", "RS384", "RS512"}),
		jwt.WithIssuedAt(),
		jwt.WithLeeway(v.leeway),
	)

	claims := jwt.MapClaims{}
	parsed, err := parser.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		return v.publicKey, nil
	})
	if err != nil || !parsed.Valid {
		return nil, errors.New("invalid token")
	}

	if v.issuer != "" {
		if !matchIssuer(claims, v.issuer) {
			return nil, errors.New("invalid token issuer")
		}
	}

	if v.audience != "" {
		if !matchAudience(claims, v.audience) {
			return nil, errors.New("invalid token audience")
		}
	}

	return claimsToContextClaims(claims), nil
}

func (v *InternalJWTVerifier) ShouldAuthenticate(fullMethod string) bool {
	if !v.Enabled() {
		return false
	}

	method := normalizeMethodName(fullMethod)

	if len(v.includeMethods) > 0 {
		_, ok := v.includeMethods[method]
		return ok
	}

	if len(v.excludeMethods) > 0 {
		_, excluded := v.excludeMethods[method]
		if excluded {
			return false
		}
	}

	return true
}

func matchIssuer(claims jwt.MapClaims, expected string) bool {
	raw, ok := claims["iss"]
	if !ok {
		return false
	}
	iss, ok := raw.(string)
	if !ok {
		return false
	}
	return strings.TrimSpace(iss) == expected
}

func matchAudience(claims jwt.MapClaims, expected string) bool {
	raw, ok := claims["aud"]
	if !ok {
		return false
	}

	switch v := raw.(type) {
	case string:
		return strings.TrimSpace(v) == expected
	case []string:
		for _, item := range v {
			if strings.TrimSpace(item) == expected {
				return true
			}
		}
	case []interface{}:
		for _, item := range v {
			s, ok := item.(string)
			if ok && strings.TrimSpace(s) == expected {
				return true
			}
		}
	}

	return false
}

func claimsToContextClaims(claims jwt.MapClaims) *Claims {
	if claims == nil {
		return nil
	}

	getString := func(keys ...string) string {
		for _, key := range keys {
			if val, ok := claims[key]; ok {
				if s, ok := val.(string); ok {
					return s
				}
			}
		}
		return ""
	}

	subject := getString("sub")
	attributes := extractAttributes(claims)

	return &Claims{
		Subject:    subject,
		SessionID:  getString("session_id", "sid"),
		Role:       getString("role"),
		Attributes: attributes,
	}
}

func extractAttributes(claims jwt.MapClaims) map[string]string {
	attrs := map[string]string{}
	raw, ok := claims["attributes"]
	if !ok || raw == nil {
		return attrs
	}

	switch v := raw.(type) {
	case map[string]string:
		for key, val := range v {
			addIfNotEmpty(attrs, key, val)
		}
	case map[string]interface{}:
		for key, val := range v {
			if s, ok := val.(string); ok {
				addIfNotEmpty(attrs, key, s)
			}
		}
	}

	return attrs
}

func parseRSAPublicKey(raw string) (*rsa.PublicKey, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.New("INTERNAL_JWT_PUBLIC_KEY is required when INTERNAL_JWT_ENABLED=true")
	}

	// Allow direct PEM content OR file path.
	if !strings.Contains(raw, "BEGIN PUBLIC KEY") {
		if b, err := os.ReadFile(raw); err == nil {
			raw = string(b)
		}
	}
	raw = strings.ReplaceAll(raw, `\n`, "\n")

	block, _ := pem.Decode([]byte(raw))
	if block == nil {
		return nil, errors.New("invalid INTERNAL_JWT_PUBLIC_KEY: not a valid PEM block")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("invalid INTERNAL_JWT_PUBLIC_KEY: %w", err)
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("invalid INTERNAL_JWT_PUBLIC_KEY: key is not RSA public key")
	}

	return rsaPub, nil
}

func normalizeLeeway(d time.Duration) time.Duration {
	if d < 0 {
		return 0
	}
	if d == 0 {
		return 30 * time.Second
	}
	return d
}

func normalizeMethodSet(in []string) map[string]struct{} {
	out := make(map[string]struct{}, len(in))
	for _, method := range in {
		m := normalizeMethodName(method)
		if m == "" {
			continue
		}
		out[m] = struct{}{}
	}
	return out
}

func normalizeMethodName(method string) string {
	m := strings.TrimSpace(method)
	if m == "" {
		return ""
	}
	if !strings.HasPrefix(m, "/") {
		return "/" + m
	}
	return m
}
