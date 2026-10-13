package grpc

import (
	"context"
	"errors"
	"strings"
	"time"

	coreErrors "github.com/yogayulanda/go-core/errors"

	"google.golang.org/grpc"
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
	return loggingInterceptorWithLogger(app.Logger(), app.Config().Log.RequestStart)
}

func inFlightInterceptor(application *app.App) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, request interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		finish := application.TrackInFlight(ctx, info.FullMethod)
		defer finish()
		return handler(ctx, request)
	}
}

func loggingInterceptorWithLogger(log logger.Logger, requestStart bool) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {

		start := time.Now()
		if requestStart {
			log.Debug(ctx, "request_started", logger.Field{Key: "app.log_type", Value: "access"}, logger.Field{Key: "app.method", Value: info.FullMethod})
		}

		resp, err := handler(ctx, req)

		duration := time.Since(start).Milliseconds()
		responseCode := "CORE-0000"
		class := "SUCCESS"
		unregisteredCode := ""
		errorMessage := ""
		if code, responseClass, original := observability.ResponseInfo(ctx); code != "" {
			responseCode = code
			class = responseClass
			unregisteredCode = original
		}

		if err != nil {
			var appErr *coreErrors.AppError
			if errors.As(err, &appErr) {
				def, registered := coreErrors.Resolve(appErr.Code)
				responseCode = def.Code
				class = string(def.Class)
				if !registered {
					unregisteredCode = appErr.Code
					errorMessage = ""
				} else {
					errorMessage = appErr.Message
				}
				err = coreErrors.ToGRPC(err)
			} else {
				if info, ok := coreErrors.InfoFromGRPC(err); ok {
					responseCode = info.Code
					class = string(info.Class)
				} else {
					responseCode = "CORE-9000"
					class = "ERROR"
					err = coreErrors.ToGRPC(err)
				}
			}
		}

		responseDef, _ := coreErrors.Resolve(responseCode)
		statusCode := responseDef.HTTPStatus
		log.LogService(ctx, logger.ServiceLog{
			LogType:           "access",
			Class:             class,
			Route:             info.FullMethod,
			Method:            info.FullMethod,
			StatusCode:        statusCode,
			ResponseCode:      responseCode,
			DurationMs:        duration,
			ErrorMessage:      errorMessage,
			UnregisteredCode:  unregisteredCode,
			RequestIDReplaced: observability.RequestIDReplaced(ctx),
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

func authInterceptorWithLogger(verifier *security.InternalJWTVerifier, log logger.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if verifier != nil && verifier.Enabled() && verifier.ShouldAuthenticate(info.FullMethod) {
			token, err := bearerTokenFromMetadata(ctx)
			if err != nil {
				logAuthResult(ctx, log, verifier, info.FullMethod, "failed", authExtractionErrorCode(err), nil)
				return nil, coreErrors.ToGRPC(coreErrors.New("CORE-1001", authClientErrorMessage()))
			}

			claims, err := verifier.Verify(token)
			if err != nil {
				logAuthResult(ctx, log, verifier, info.FullMethod, "failed", security.AuthErrorCode(err), map[string]interface{}{
					"issuer_set":   verifier.ConfigMetadata()["issuer_set"],
					"audience_set": verifier.ConfigMetadata()["audience_set"],
				})
				return nil, coreErrors.ToGRPC(coreErrors.New("CORE-1001", authClientErrorMessage()))
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
			logAuthResult(ctx, log, verifier, info.FullMethod, "success", "", security.ExtractMetadataSummary(ctx))
		}

		return handler(ctx, req)
	}
}

func bearerTokenFromMetadata(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", errMissingMetadata
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		return "", errMissingAuthorization
	}

	raw := strings.TrimSpace(values[0])
	if raw == "" {
		return "", errAuthorizationHeaderEmpty
	}

	parts := strings.SplitN(raw, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", errInvalidAuthorization
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", errAuthorizationTokenEmpty
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

		ctx = observability.EnsureFieldBag(ctx)
		md, _ := metadata.FromIncomingContext(ctx)
		requestIDs := md.Get("x-request-id")
		requestID := ""
		if len(requestIDs) > 0 && observability.IsValidRequestID(requestIDs[0]) {
			requestID = requestIDs[0]
		} else {
			requestID = observability.GenerateRequestID()
			if len(requestIDs) > 0 {
				observability.MarkRequestIDReplaced(ctx)
			}
		}
		ctx = observability.WithRequestID(ctx, requestID)
		_ = grpc.SetHeader(ctx, metadata.Pairs("x-request-id", requestID))

		userIDs := md.Get("x-user-id")
		labels := make(map[string]string)
		metadataValues := map[string]struct{}{}
		for _, label := range observability.PropagatedLabels() {
			metadataValues[label] = struct{}{}
		}
		for key, values := range md {
			if !strings.HasPrefix(key, observability.LabelMetadataPrefix) || len(values) == 0 {
				continue
			}
			label := strings.TrimPrefix(key, observability.LabelMetadataPrefix)
			if _, registered := metadataValues[label]; registered {
				labels[label] = values[0]
			}
		}
		userID := ""
		if len(userIDs) > 0 {
			userID = userIDs[0]
		}
		observability.RestoreIncomingIdentity(ctx, userID, labels)
		if values := md.Get("x-core-client-ip"); len(values) > 0 {
			_ = observability.SetClientIP(ctx, values[0])
		}
		if values := md.Get("x-core-user-agent"); len(values) > 0 {
			_ = observability.SetUserAgent(ctx, values[0])
		}
		if values := md.Get("x-core-request-id-replaced"); len(values) > 0 && values[0] == "true" {
			observability.MarkRequestIDReplaced(ctx)
		}

		return handler(ctx, req)
	}
}
