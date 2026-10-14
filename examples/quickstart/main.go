package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/yogayulanda/go-core/app"
	"github.com/yogayulanda/go-core/config"
	"github.com/yogayulanda/go-core/server"
	coregateway "github.com/yogayulanda/go-core/server/gateway"
	coregrpc "github.com/yogayulanda/go-core/server/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load(config.WithDotEnv(".env"))
	if err != nil {
		log.Fatal(err)
	}
	if err := cfg.Validate(); err != nil {
		log.Fatal(err)
	}

	application, err := app.New(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}
	grpcServer, err := coregrpc.New(application)
	if err != nil {
		log.Fatal(err)
	}

	connection, err := grpc.NewClient(
		"127.0.0.1:"+portString(cfg.GRPC.Port),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer connection.Close()
	healthClient := grpc_health_v1.NewHealthClient(connection)

	gatewayServer, err := coregateway.New(application, func(_ context.Context, mux *runtime.ServeMux) error {
		return mux.HandlePath(http.MethodGet, "/ping", func(w http.ResponseWriter, r *http.Request, _ map[string]string) {
			result, err := healthClient.Check(r.Context(), &grpc_health_v1.HealthCheckRequest{})
			if err != nil {
				http.Error(w, "gRPC ping failed", http.StatusBadGateway)
				return
			}
			if result.Status != grpc_health_v1.HealthCheckResponse_SERVING {
				http.Error(w, "gRPC service is not serving", http.StatusServiceUnavailable)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "pong"})
		})
	})
	if err != nil {
		log.Fatal(err)
	}

	if err := server.Run(ctx, application, grpcServer, gatewayServer); err != nil {
		log.Fatal(err)
	}
}

func portString(port int) string {
	return strconv.Itoa(port)
}
