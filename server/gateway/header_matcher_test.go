package gateway

import "testing"

func TestIncomingHeaderMatcherForwardsBatchCIdentity(t *testing.T) {
	for _, header := range []string{"X-Request-ID", "X-User-ID", clientIPHeader, userAgentHeader, requestIDReplacedHeader, "X-Label-Client-App"} {
		got, ok := incomingHeaderMatcher(header)
		if !ok || got == "" {
			t.Errorf("header %q not forwarded", header)
		}
	}
}

func TestOutgoingHeaderMatcherMapsAndFiltersInternalHeaders(t *testing.T) {
	for input, expected := range map[string]string{
		"x-request-id":                  "X-Request-ID",
		"x-response-code":               "X-Response-Code",
		"Grpc-Metadata-X-Response-Code": "X-Response-Code",
	} {
		got, ok := outgoingHeaderMatcher(input)
		if !ok || got != expected {
			t.Errorf("outgoing matcher(%q) = %q, %v; want %q, true", input, got, ok, expected)
		}
	}
}

func TestResolveClientIPUsesConfiguredRightmostProxyCount(t *testing.T) {
	forwarded := "1.1.1.1, 10.0.0.2"
	if got := resolveClientIP("10.0.0.3:443", forwarded, 1); got != "1.1.1.1" {
		t.Fatalf("trustedProxyHops=1: expected 1.1.1.1, got %q", got)
	}
	if got := resolveClientIP("10.0.0.3:443", forwarded, 0); got != "10.0.0.3" {
		t.Fatalf("trustedProxyHops=0: expected connection address, got %q", got)
	}
}
