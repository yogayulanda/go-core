package grpc

import (
	"context"
	"errors"

	"github.com/yogayulanda/go-core/logger"
	"github.com/yogayulanda/go-core/observability"
	"github.com/yogayulanda/go-core/security"
)

var (
	errMissingMetadata          = errors.New("missing metadata")
	errMissingAuthorization     = errors.New("missing authorization header")
	errAuthorizationHeaderEmpty = errors.New("authorization header is empty")
	errInvalidAuthorization     = errors.New("invalid authorization scheme")
	errAuthorizationTokenEmpty  = errors.New("authorization token is empty")
)

func authClientErrorMessage() string {
	return "unauthorized request"
}

func authExtractionErrorCode(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, errMissingMetadata):
		return "missing_metadata"
	case errors.Is(err, errMissingAuthorization):
		return "missing_authorization_header"
	case errors.Is(err, errAuthorizationHeaderEmpty):
		return "authorization_header_empty"
	case errors.Is(err, errInvalidAuthorization):
		return "invalid_authorization_scheme"
	case errors.Is(err, errAuthorizationTokenEmpty):
		return "authorization_token_empty"
	default:
		return "auth_extract_failed"
	}
}

func logAuthConfig(ctx context.Context, log logger.Logger, verifier *security.InternalJWTVerifier) {
	if log == nil {
		return
	}
	log.LogService(ctx, logger.ServiceLog{
		Operation: "auth_config",
		Status:    "configured",
		Metadata:  verifier.ConfigMetadata(),
	})
}

func logAuthResult(
	ctx context.Context,
	log logger.Logger,
	verifier *security.InternalJWTVerifier,
	fullMethod string,
	status string,
	errorCode string,
	metadata map[string]interface{},
) {
	if log == nil {
		return
	}

	base := map[string]interface{}{
		"auth_mode":    verifier.AuthMode(),
		"policy_mode":  verifier.PolicyMode(),
		"method":       fullMethod,
		"request_id":   observability.GetRequestID(ctx),
		"jwt_enforced": verifier.Enabled() && verifier.ShouldAuthenticate(fullMethod),
	}
	for k, v := range metadata {
		base[k] = v
	}

	log.LogService(ctx, logger.ServiceLog{
		Operation: "auth_request",
		Status:    status,
		ErrorCode: errorCode,
		Metadata:  base,
	})
}
