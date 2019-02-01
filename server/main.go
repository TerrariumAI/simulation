package main

import (
	"log"
	"net"

	pb "github.com/olamai/proto"
	context "golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type vec2 struct {
	X int32
	Y int32
}

type Entity struct {
	Class string
	Pos   vec2
}

// Server represents the gRPC server
type Server struct {
	agents       []*Entity
	chSpawnAgent chan pb.SpawnAgentMessage
}

func main() {
	var simulationServer = Server{chSpawnAgent: make(chan pb.SpawnAgentMessage, 3)}

	// Start the simulation on another thread
	go simulationServer.startSimulation()

	lis, err := net.Listen("tcp", ":7771")
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

func (s *Server) SpawnAgent(ctx context.Context, in *pb.SpawnAgentMessage) (*pb.SpawnAgentResultMessage, error) {
	log.Printf("Receive spawn Entity message %s", in.X)
	log.Printf("Receive spawn Entity message %s", in.Y)
	s.chSpawnAgent <- *in
	return &pb.SpawnAgentResultMessage{Status: "success"}, nil
}
