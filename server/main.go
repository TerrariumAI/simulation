package main

import (
	"log"
	"net"
	"os"

	. "github.com/olamai/proto/simulation"
	"google.golang.org/grpc"
)

type Server struct {
	world World
	env   string
}

func main() {
	// Get the ENV from environment variable, or default to dev
	env := os.Getenv("ENV")
	if env == "" {
		env = "dev"
	}

	// Initialize server obj
	var simulationServer = Server{
		world: NewWorld(),
		env:   env,
	}

	// Works with envoy hosting at 0.0.0.0:9090
	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	RegisterSimulationServer(grpcServer, &simulationServer)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
