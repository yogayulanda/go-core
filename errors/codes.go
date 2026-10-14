package errors

import (
	"context"
	"fmt"
	"sync"

	"google.golang.org/grpc/codes"
)

type Class string

const (
	ClassSuccess Class = "SUCCESS"
	ClassPending Class = "PENDING"
	ClassUnknown Class = "UNKNOWN"
	ClassFailed  Class = "FAILED"
	ClassError   Class = "ERROR"
)

type Finality string

const (
	FinalityBusiness                Finality = "Business"
	FinalityTechnicalRecoverable    Finality = "Technical Recoverable"
	FinalityTechnicalNonRecoverable Finality = "Technical Non-Recoverable"
	FinalityAmbiguous               Finality = "Ambiguous"
)

type Def struct {
	Code           string
	Class          Class
	HTTPStatus     int
	GRPCCode       codes.Code
	Finality       Finality
	Retryable      bool
	DefaultMessage string
}

type MessageProvider interface {
	Message(ctx context.Context, code string) (string, bool)
}

var registry = struct {
	sync.RWMutex
	definitions     map[string]Def
	messageProvider MessageProvider
}{definitions: make(map[string]Def)}

type RegistryState struct {
	definitions     map[string]Def
	messageProvider MessageProvider
}

func SnapshotRegistry() RegistryState {
	registry.RLock()
	defer registry.RUnlock()
	definitions := make(map[string]Def, len(registry.definitions))
	for code, definition := range registry.definitions {
		definitions[code] = definition
	}
	return RegistryState{definitions: definitions, messageProvider: registry.messageProvider}
}

func RestoreRegistry(state RegistryState) {
	registry.Lock()
	defer registry.Unlock()
	registry.definitions = make(map[string]Def, len(state.definitions))
	for code, definition := range state.definitions {
		registry.definitions[code] = definition
	}
	registry.messageProvider = state.messageProvider
}

func init() {
	_ = Register(
		Def{Code: "CORE-0000", Class: ClassSuccess, HTTPStatus: 200, GRPCCode: codes.OK, Finality: FinalityBusiness, DefaultMessage: "success"},
		Def{Code: "CORE-1000", Class: ClassFailed, HTTPStatus: 400, GRPCCode: codes.InvalidArgument, Finality: FinalityBusiness, DefaultMessage: "invalid request"},
		Def{Code: "CORE-1001", Class: ClassFailed, HTTPStatus: 401, GRPCCode: codes.Unauthenticated, Finality: FinalityBusiness, DefaultMessage: "unauthorized request"},
		Def{Code: "CORE-1002", Class: ClassFailed, HTTPStatus: 401, GRPCCode: codes.Unauthenticated, Finality: FinalityBusiness, DefaultMessage: "session expired"},
		Def{Code: "CORE-1003", Class: ClassFailed, HTTPStatus: 403, GRPCCode: codes.PermissionDenied, Finality: FinalityBusiness, DefaultMessage: "forbidden"},
		Def{Code: "CORE-1004", Class: ClassFailed, HTTPStatus: 404, GRPCCode: codes.NotFound, Finality: FinalityBusiness, DefaultMessage: "not found"},
		Def{Code: "CORE-1005", Class: ClassFailed, HTTPStatus: 413, GRPCCode: codes.ResourceExhausted, Finality: FinalityBusiness, DefaultMessage: "request body too large"},
		Def{Code: "CORE-9000", Class: ClassError, HTTPStatus: 500, GRPCCode: codes.Internal, Finality: FinalityTechnicalNonRecoverable, DefaultMessage: "internal server error"},
		Def{Code: "CORE-9001", Class: ClassError, HTTPStatus: 503, GRPCCode: codes.Unavailable, Finality: FinalityTechnicalRecoverable, Retryable: true, DefaultMessage: "service unavailable"},
	)
}

func Register(defs ...Def) error {
	registry.Lock()
	defer registry.Unlock()

	seen := make(map[string]struct{}, len(defs))
	for _, def := range defs {
		if err := validateDef(def); err != nil {
			return err
		}
		if _, ok := registry.definitions[def.Code]; ok {
			return fmt.Errorf("response code %q is already registered", def.Code)
		}
		if _, ok := seen[def.Code]; ok {
			return fmt.Errorf("response code %q is duplicated in registration", def.Code)
		}
		seen[def.Code] = struct{}{}
	}
	for _, def := range defs {
		registry.definitions[def.Code] = def
	}
	return nil
}

func Lookup(code string) (Def, bool) {
	registry.RLock()
	def, ok := registry.definitions[code]
	registry.RUnlock()
	return def, ok
}

func Resolve(code string) (Def, bool) {
	if def, ok := Lookup(code); ok {
		return def, true
	}
	def, _ := Lookup("CORE-9000")
	return def, false
}

func SetMessageProvider(provider MessageProvider) {
	registry.Lock()
	registry.messageProvider = provider
	registry.Unlock()
}

func Message(ctx context.Context, code string) string {
	def, _ := Resolve(code)
	return messageFor(ctx, def.Code, def)
}

func messageFor(ctx context.Context, code string, def Def) string {
	registry.RLock()
	provider := registry.messageProvider
	registry.RUnlock()
	if provider != nil {
		if message, ok := provider.Message(ctx, code); ok && message != "" {
			return message
		}
	}
	return def.DefaultMessage
}

func validateDef(def Def) error {
	if def.Code == "" {
		return fmt.Errorf("response code must not be empty")
	}
	switch def.Class {
	case ClassSuccess, ClassPending, ClassUnknown, ClassFailed, ClassError:
	default:
		return fmt.Errorf("response code %q has invalid class %q", def.Code, def.Class)
	}
	if def.HTTPStatus < 100 || def.HTTPStatus > 599 {
		return fmt.Errorf("response code %q has invalid HTTP status %d", def.Code, def.HTTPStatus)
	}
	if def.GRPCCode < codes.OK || def.GRPCCode > codes.Unauthenticated {
		return fmt.Errorf("response code %q has invalid gRPC code %d", def.Code, def.GRPCCode)
	}
	switch def.Finality {
	case FinalityBusiness, FinalityTechnicalRecoverable, FinalityTechnicalNonRecoverable, FinalityAmbiguous:
	default:
		return fmt.Errorf("response code %q has invalid finality %q", def.Code, def.Finality)
	}
	if def.DefaultMessage == "" {
		return fmt.Errorf("response code %q must have a default message", def.Code)
	}
	return nil
}
