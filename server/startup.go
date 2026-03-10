package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/yogayulanda/go-core/logger"
	annotations "google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// DescribeFromProto returns sorted HTTP endpoint mappings and gRPC methods.
func DescribeFromProto(
	fd protoreflect.FileDescriptor,
	includeCoreHTTPRoutes bool,
) (httpEndpoints []string, grpcMethods []string) {
	httpSet := map[string]struct{}{}
	grpcSet := map[string]struct{}{}

	services := fd.Services()
	for i := 0; i < services.Len(); i++ {
		svc := services.Get(i)
		svcFQN := fmt.Sprintf("/%s/%s", string(fd.Package()), svc.Name())

		methods := svc.Methods()
		for j := 0; j < methods.Len(); j++ {
			m := methods.Get(j)
			grpcSet[fmt.Sprintf("%s/%s", svcFQN, m.Name())] = struct{}{}

			opts, ok := m.Options().(*descriptorpb.MethodOptions)
			if !ok || opts == nil {
				continue
			}

			ext := proto.GetExtension(opts, annotations.E_Http)
			httpRule, ok := ext.(*annotations.HttpRule)
			if !ok || httpRule == nil {
				continue
			}

			addHTTPRule(httpSet, httpRule)
		}
	}

	if includeCoreHTTPRoutes {
		httpSet["GET /health"] = struct{}{}
		httpSet["GET /ready"] = struct{}{}
		httpSet["GET /version"] = struct{}{}
		httpSet["GET /metrics"] = struct{}{}
	}

	httpEndpoints = setToSortedSlice(httpSet)
	grpcMethods = setToSortedSlice(grpcSet)
	return httpEndpoints, grpcMethods
}

// LogStartupReadiness logs startup readiness for gRPC and HTTP gateway.
// For TLS-enabled gateway, readiness is checked at TCP transport level.
func LogStartupReadiness(
	ctx context.Context,
	log logger.Logger,
	grpcPort int,
	httpPort int,
	timeout time.Duration,
	httpTLSEnabled bool,
) {
	grpcReady := waitForTCP(ctx, grpcPort, timeout)
	if grpcReady {
		log.Info(context.Background(), "gRPC server ready",
			logger.Field{Key: "grpc_port", Value: grpcPort},
		)
	} else {
		log.Warn(context.Background(), "gRPC readiness check timeout",
			logger.Field{Key: "grpc_port", Value: grpcPort},
		)
	}

	gatewayReady := waitForHTTPHealth(ctx, httpPort, timeout, httpTLSEnabled)
	if gatewayReady {
		healthScheme := "http"
		if httpTLSEnabled {
			healthScheme = "https"
		}
		log.Info(context.Background(), "HTTP gateway ready",
			logger.Field{Key: "http_port", Value: httpPort},
			logger.Field{Key: "health", Value: fmt.Sprintf("%s://127.0.0.1:%d/health", healthScheme, httpPort)},
		)
	} else {
		log.Warn(context.Background(), "HTTP gateway readiness check timeout",
			logger.Field{Key: "http_port", Value: httpPort},
		)
	}

	if grpcReady && gatewayReady {
		log.Info(context.Background(), "service ready",
			logger.Field{Key: "grpc_port", Value: grpcPort},
			logger.Field{Key: "http_port", Value: httpPort},
		)
	}
}

func addHTTPRule(set map[string]struct{}, rule *annotations.HttpRule) {
	method, path := extractMethodPath(rule)
	if method != "" && path != "" {
		set[fmt.Sprintf("%s %s", method, path)] = struct{}{}
	}
	for _, extra := range rule.AdditionalBindings {
		m, p := extractMethodPath(extra)
		if m != "" && p != "" {
			set[fmt.Sprintf("%s %s", m, p)] = struct{}{}
		}
	}
}

func extractMethodPath(rule *annotations.HttpRule) (method, path string) {
	switch pat := rule.Pattern.(type) {
	case *annotations.HttpRule_Get:
		return http.MethodGet, normalizePath(pat.Get)
	case *annotations.HttpRule_Post:
		return http.MethodPost, normalizePath(pat.Post)
	case *annotations.HttpRule_Put:
		return http.MethodPut, normalizePath(pat.Put)
	case *annotations.HttpRule_Delete:
		return http.MethodDelete, normalizePath(pat.Delete)
	case *annotations.HttpRule_Patch:
		return http.MethodPatch, normalizePath(pat.Patch)
	case *annotations.HttpRule_Custom:
		if pat.Custom == nil {
			return "", ""
		}
		return strings.ToUpper(strings.TrimSpace(pat.Custom.Kind)), normalizePath(pat.Custom.Path)
	default:
		return "", ""
	}
}

func normalizePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "/") {
		return path
	}
	return "/" + path
}

func setToSortedSlice(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for item := range set {
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func waitForTCP(ctx context.Context, port int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	address := fmt.Sprintf("127.0.0.1:%d", port)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return false
		default:
		}

		conn, err := net.DialTimeout("tcp", address, 400*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return true
		}
		time.Sleep(200 * time.Millisecond)
	}

	return false
}

func waitForHTTPHealth(ctx context.Context, port int, timeout time.Duration, tlsEnabled bool) bool {
	if tlsEnabled {
		// TLS endpoint readiness is confirmed at transport level.
		// HTTP-level probe cannot assume certificate trust context safely.
		return waitForTCP(ctx, port, timeout)
	}

	deadline := time.Now().Add(timeout)
	httpURL := fmt.Sprintf("http://127.0.0.1:%d/health", port)

	httpClient := &http.Client{Timeout: 800 * time.Millisecond}

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return false
		default:
		}

		if checkHTTPHealth(ctx, httpClient, httpURL) {
			return true
		}

		time.Sleep(200 * time.Millisecond)
	}

	return false
}

func checkHTTPHealth(ctx context.Context, client *http.Client, url string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	_ = resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
