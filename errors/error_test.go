package errors

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"google.golang.org/grpc/codes"
)

func TestRegisterRejectsDuplicateAndInvalidDefinitions(t *testing.T) {
	if err := Register(Def{Code: "CORE-0000", Class: ClassSuccess, HTTPStatus: 200, GRPCCode: codes.OK, Finality: FinalityBusiness, DefaultMessage: "duplicate"}); err == nil {
		t.Fatal("expected built-in duplicate registration error")
	}
	if err := Register(Def{Code: "", Class: ClassSuccess, HTTPStatus: 200, GRPCCode: codes.OK, Finality: FinalityBusiness, DefaultMessage: "invalid"}); err == nil {
		t.Fatal("expected invalid definition error")
	}
}

func TestResolveUnknownUsesInternalFallback(t *testing.T) {
	def, registered := Resolve("NOT-REGISTERED")
	if registered {
		t.Fatal("unknown code was marked registered")
	}
	if def.Code != "CORE-9000" || def.HTTPStatus != 500 || def.GRPCCode != codes.Internal {
		t.Fatalf("unexpected fallback definition: %+v", def)
	}
}

func TestLookupConcurrentReads(t *testing.T) {
	const code = "TEST-LOOKUP-CONCURRENT"
	if err := Register(Def{Code: code, Class: ClassFailed, HTTPStatus: 422, GRPCCode: codes.InvalidArgument, Finality: FinalityBusiness, DefaultMessage: "failed"}); err != nil {
		t.Fatalf("register test definition: %v", err)
	}
	var workers sync.WaitGroup
	for range 20 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for range 100 {
				if def, ok := Lookup(code); !ok || def.HTTPStatus != 422 {
					t.Errorf("concurrent lookup returned %+v, %v", def, ok)
					return
				}
			}
		}()
	}
	workers.Wait()
}

func TestAppErrorUnwrapAndError(t *testing.T) {
	inner := errors.New("database detail")
	appErr := Wrap("CORE-9001", "service unavailable", inner)
	if !errors.Is(appErr, inner) {
		t.Fatal("expected wrapped internal error to be discoverable")
	}
	if got := appErr.Error(); got != "CORE-9001: service unavailable" {
		t.Fatalf("unexpected error string: %s", got)
	}
}

func TestAppErrorCanWrapWithContext(t *testing.T) {
	appErr := fmt.Errorf("outer: %w", Validation("invalid input", Detail{Field: "amount", Reason: "must be positive"}))
	var got *AppError
	if !errors.As(appErr, &got) || got.Code != "CORE-1000" {
		t.Fatalf("expected wrapped validation AppError, got %v", appErr)
	}
}
