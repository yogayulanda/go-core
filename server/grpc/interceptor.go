package grpc

import (
	"context"
	"errors"
	"strings"
	"time"

	coreErrors "github.com/yogayulanda/go-core/errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/yogayulanda/go-core/app"
	"github.com/yogayulanda/go-core/logger"
	"github.com/yogayulanda/go-core/observability"
	"github.com/yogayulanda/go-core/security"
)

func recoveryInterceptor(log logger.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {

		defer func() {
			if r := recover(); r != nil {
				log.LogService(ctx, logger.ServiceLog{
					Operation: "grpc_request",
					Status:    "failed",
					ErrorCode: "panic_recovered",
					Metadata: map[string]interface{}{
						"method": info.FullMethod,
						"panic":  r,
					},
				})
				err = status.Error(13, "internal server error")
			}
		}()

		return handler(ctx, req)
	}
}

func loggingInterceptor(app *app.App) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {

		start := time.Now()

		resp, err := handler(ctx, req)

		duration := time.Since(start).Milliseconds()
		statusStr := "success"
		errorCode := ""
		errorCategory := ""

		if err != nil {
			statusStr = "failed"

			// Map back to AppError if possible
			var appErr *coreErrors.AppError
			if errors.As(err, &appErr) {
				errorCode = string(appErr.Code)
				errorCategory = string(appErr.Category)
				err = coreErrors.ToGRPC(appErr)
			} else {
				errorCode = string(coreErrors.CodeInternal)
				errorCategory = string(coreErrors.CategoryInternal)
				err = coreErrors.ToGRPC(err)
			}
		}

		requestID := observability.GetRequestID(ctx)

		app.Logger().LogService(ctx, logger.ServiceLog{
			Operation:  "grpc_request",
			Status:     statusStr,
			DurationMs: duration,
			ErrorCode:  errorCode,
			Metadata: map[string]interface{}{
				"method":         info.FullMethod,
				"error_category": errorCategory,
				"request_id":     requestID,
			},
		})

		return resp, err
	}
}

func metricsInterceptor(app *app.App) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {

		start := time.Now()

		resp, err := handler(ctx, req)

		duration := time.Since(start).Seconds()

		statusStr := "success"
		if err != nil {
			statusStr = "failed"
		}

		serviceName := app.Config().App.ServiceName

		app.Metrics().RequestTotal.WithLabelValues(
			serviceName,
			info.FullMethod,
			statusStr,
		).Inc()

		app.Metrics().RequestDuration.WithLabelValues(
			serviceName,
			info.FullMethod,
		).Observe(duration)

		app.Metrics().ServiceTotal.WithLabelValues(
			serviceName,
			"grpc_request",
			statusStr,
		).Inc()

		app.Metrics().ServiceDuration.WithLabelValues(
			serviceName,
			"grpc_request",
		).Observe(duration)

		return resp, err
	}
}

func authInterceptor(verifier *security.InternalJWTVerifier) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if verifier != nil && verifier.Enabled() && verifier.ShouldAuthenticate(info.FullMethod) {
			token, err := bearerTokenFromMetadata(ctx)
			if err != nil {
				return nil, status.Error(codes.Unauthenticated, err.Error())
			}

			claims, err := verifier.Verify(token)
			if err != nil {
				return nil, status.Error(codes.Unauthenticated, err.Error())
			}

			if claims != nil {
				ctx = security.Inject(ctx, claims)
			}

			return handler(ctx, req)
		}

		if verifier == nil || !verifier.Enabled() {
			claims := security.ExtractFromMetadata(ctx)
			if claims != nil {
				ctx = security.Inject(ctx, claims)
			}
		}

		return handler(ctx, req)
	}
}

func bearerTokenFromMetadata(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", errors.New("missing metadata")
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		return "", errors.New("missing authorization header")
	}

	raw := strings.TrimSpace(values[0])
	if raw == "" {
		return "", errors.New("authorization header is empty")
	}

	parts := strings.SplitN(raw, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", errors.New("invalid authorization scheme")
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", errors.New("authorization token is empty")
	}

	return token, nil
}

func requestIDInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {

		md, ok := metadata.FromIncomingContext(ctx)

		var requestID string

		if ok {
			values := md.Get("x-request-id")
			if len(values) > 0 {
				requestID = values[0]
			}
		}

		if requestID == "" {
			requestID = observability.GenerateRequestID()
		}

		ctx = observability.WithRequestID(ctx, requestID)

		return handler(ctx, req)
	}
}
