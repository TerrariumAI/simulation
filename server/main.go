package main

import (
	"log"
	"net"

	pb "github.com/olamai/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	world World
}

func main() {
	// Initialize server obj
	var simulationServer = Server{
		world: World{
			entities:              make(map[string]*Entity),
			posEntityMatrix:       make(map[Vec2]*Entity),
			observerationChannels: make(map[string]chan pb.CellUpdate),
		},
	}

	// Works with envoy hosting at 0.0.0.0:9090
	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterSimulationServer(grpcServer, &simulationServer)
	reflection.Register(grpcServer)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
