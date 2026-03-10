package security

import "context"

type Claims struct {
	Subject    string
	SessionID  string
	Role       string
	Attributes map[string]string
}

type contextKey string

const claimsKey contextKey = "security_claims"

func Inject(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

func FromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(claimsKey).(*Claims)
	return claims, ok
}
