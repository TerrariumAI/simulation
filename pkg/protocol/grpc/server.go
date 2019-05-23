package grpc

import (
	"context"
	"net"
	"os"
	"os/signal"

	v1 "github.com/terrariumai/simulation/pkg/api/v1"
	"github.com/terrariumai/simulation/pkg/logger"
	"github.com/terrariumai/simulation/pkg/protocol/grpc/middleware"
	"google.golang.org/grpc"
)

// RunServer runs gRPC service to publish Simulation service
func RunServer(ctx context.Context, v1API v1.SimulationServer, port string) error {
	listen, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}

	// gRPC server statup options
	// opts := []grpc.ServerOption{
	// 	grpc.KeepaliveParams(
	// 		keepalive.ServerParameters{
	// 			Time:    (time.Duration(2) * time.Second),
	// 			Timeout: (time.Duration(2) * time.Second),
	// 		},
	// 	),
	// 	grpc.KeepaliveEnforcementPolicy(
	// 		keepalive.EnforcementPolicy{
	// 			MinTime:             (time.Duration(2) * time.Second),
	// 			PermitWithoutStream: false,
	// 		},
	// 	),
	// }

	opts := []grpc.ServerOption{}
	// add middleware
	opts = middleware.AddLogging(logger.Log, opts)

	// register service
	server := grpc.NewServer(opts...)
	v1.RegisterSimulationServer(server, v1API)

	// graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			// sig is a ^C, handle it
			logger.Log.Warn("shutting down gRPC server...")

			// Not graceful stop because Spectate RPCs will never complete
			server.Stop()

			logger.Log.Warn("grpc server shut down!")

			<-ctx.Done()
		}
	}()

	// start gRPC server
	logger.Log.Info("starting gRPC server...")
	return server.Serve(listen)
}
