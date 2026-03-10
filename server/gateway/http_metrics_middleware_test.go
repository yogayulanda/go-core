package gateway

import "testing"

func TestNormalizeHTTPRoute(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "root", in: "/", want: "/"},
		{name: "health", in: "/health", want: "/health"},
		{name: "numeric id", in: "/v1/history/123", want: "/v1/history/:id"},
		{name: "uuid id", in: "/v1/history/90fb72ed-44a8-4836-af08-8979ed578ef8", want: "/v1/history/:id"},
		{name: "mixed dynamic", in: "/v1/history/seed-tx-001", want: "/v1/history/:param"},
		{name: "long token", in: "/v1/history/abcdefghijklmnopqrstuvwxyz123456", want: "/v1/history/:param"},
		{name: "stable path", in: "/v1/history", want: "/v1/history"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeHTTPRoute(tt.in)
			if got != tt.want {
				t.Fatalf("normalizeHTTPRoute(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestIsLikelyDynamicSegment(t *testing.T) {
	if !isLikelyDynamicSegment("abc12345") {
		t.Fatalf("expected mixed alnum segment as dynamic")
	}
	if !isLikelyDynamicSegment("very-long-token-value-001") {
		t.Fatalf("expected dashed token segment as dynamic")
	}
	if isLikelyDynamicSegment("history") {
		t.Fatalf("expected stable segment not dynamic")
	}
}
