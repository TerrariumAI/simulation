package main

import (
	"log"
	"net"

	pb "github.com/olamai/proto"
	context "golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// Server represents the gRPC server
type Server struct {
}

func main() {
	lis, err := net.Listen("tcp", ":7771")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterSimulationServer(s, &Server{})
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

func (s *Server) SpawnAgent(ctx context.Context, in *pb.SpawnAgentMessage) (*pb.SpawnAgentResultMessage, error) {
	log.Printf("Receive spawn agent message %s", in.X)
	log.Printf("Receive spawn agent message %s", in.Y)
	return &pb.SpawnAgentResultMessage{Status: "success"}, nil
}
