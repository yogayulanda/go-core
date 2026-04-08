package examples

import (
	"context"

	coreapp "github.com/yogayulanda/go-core/app"
	coreconfig "github.com/yogayulanda/go-core/config"
	coremigration "github.com/yogayulanda/go-core/migration"
	coreserver "github.com/yogayulanda/go-core/server"
	coregateway "github.com/yogayulanda/go-core/server/gateway"
	coregrpc "github.com/yogayulanda/go-core/server/grpc"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

// BootstrapExample shows the canonical startup flow for a service using go-core.
func BootstrapExample(
	ctx context.Context,
	registerGRPC func(*grpc.Server),
	registerGateway func(context.Context, *runtime.ServeMux) error,
) error {
	cfg, err := coreconfig.Load()
	if err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	if err := coremigration.AutoRunUp(cfg); err != nil {
		return err
	}

	application, err := coreapp.New(ctx, cfg)
	if err != nil {
		return err
	}

	grpcServer, err := coregrpc.New(application)
	if err != nil {
		return err
	}
	gatewayServer, err := coregateway.New(application, registerGateway)
	if err != nil {
		return err
	}

	if registerGRPC != nil {
		grpcServer.Register(registerGRPC)
	}

	return coreserver.Run(ctx, application, grpcServer, gatewayServer)
}
