package grpc

import "testing"

func TestNew_NilApplication_ReturnError(t *testing.T) {
	_, err := New(nil)
	if err == nil {
		t.Fatalf("expected error for nil application")
	}
}
