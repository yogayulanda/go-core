package observability

import "context"

// WithGTID injects the money transaction identifier into the request field bag.
func WithGTID(ctx context.Context, gtid string) context.Context {
	ctx = EnsureFieldBag(ctx)
	_ = SetLabel(ctx, "gtid", gtid)
	return ctx
}

// GetGTID extracts the money transaction identifier from the request field bag.
func GetGTID(ctx context.Context) string {
	return Labels(ctx)["gtid"]
}
