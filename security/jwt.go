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

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/yogayulanda/go-core/config"
)

var (
	errAuthorizationTokenEmpty = errors.New("authorization token is empty")
	errInvalidToken            = errors.New("invalid token")
	errInvalidTokenIssuer      = errors.New("invalid token issuer")
	errInvalidTokenAudience    = errors.New("invalid token audience")
)

type InternalJWTVerifier struct {
	enabled        bool
	keyFunc        jwt.Keyfunc
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

	if strings.TrimSpace(cfg.JWKSEndpoint) != "" {
		// v3 keyfunc fetching
		kf, err := keyfunc.NewDefault([]string{cfg.JWKSEndpoint})
		if err != nil {
			return nil, fmt.Errorf("failed to initialize JWKS from %s: %w", cfg.JWKSEndpoint, err)
		}
		verifier.keyFunc = kf.Keyfunc
		return verifier, nil
	}

	key, err := parseRSAPublicKey(cfg.PublicKey)
	if err != nil {
		return nil, err
	}
	// Fallback to static public key
	verifier.keyFunc = func(token *jwt.Token) (interface{}, error) {
		return key, nil
	}

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
		return nil, errAuthorizationTokenEmpty
	}

	parser := jwt.NewParser(
		jwt.WithIssuedAt(),
		jwt.WithLeeway(v.leeway),
	)

	claims := jwt.MapClaims{}
	parsed, err := parser.ParseWithClaims(token, claims, v.keyFunc)
	if err != nil || !parsed.Valid {
		return nil, errInvalidToken
	}

	if v.issuer != "" {
		if !matchIssuer(claims, v.issuer) {
			return nil, errInvalidTokenIssuer
		}
	}

	if v.audience != "" {
		if !matchAudience(claims, v.audience) {
			return nil, errInvalidTokenAudience
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

func (v *InternalJWTVerifier) AuthMode() string {
	if v != nil && v.Enabled() {
		return "jwt"
	}
	return "metadata"
}

func (v *InternalJWTVerifier) PolicyMode() string {
	if v == nil || !v.Enabled() {
		return "metadata"
	}
	if len(v.includeMethods) > 0 {
		return "include"
	}
	if len(v.excludeMethods) > 0 {
		return "exclude"
	}
	return "all"
}

func (v *InternalJWTVerifier) ConfigMetadata() map[string]interface{} {
	if v == nil {
		return map[string]interface{}{
			"auth_mode":    "metadata",
			"policy_mode":  "metadata",
			"jwt_enabled":  false,
			"issuer_set":   false,
			"audience_set": false,
		}
	}

	return map[string]interface{}{
		"auth_mode":            v.AuthMode(),
		"policy_mode":          v.PolicyMode(),
		"jwt_enabled":          v.Enabled(),
		"issuer_set":           v.issuer != "",
		"audience_set":         v.audience != "",
		"include_method_count": len(v.includeMethods),
		"exclude_method_count": len(v.excludeMethods),
		"leeway_ms":            v.leeway.Milliseconds(),
	}
}

func AuthErrorCode(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, errAuthorizationTokenEmpty):
		return "authorization_token_empty"
	case errors.Is(err, errInvalidToken):
		return "invalid_token"
	case errors.Is(err, errInvalidTokenIssuer):
		return "invalid_token_issuer"
	case errors.Is(err, errInvalidTokenAudience):
		return "invalid_token_audience"
	default:
		return "auth_failed"
	}
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
