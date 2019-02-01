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
	pb.RegisterDatacomServer(s, &Server{})
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

func (s *Server) Execute(ctx context.Context, in *pb.ExecuteMessage) (*pb.Empty, error) {
	log.Printf("Receive execution %s", in.Execution)
	return &pb.Empty{}, nil
}

func (s *Server) Query(ctx context.Context, in *pb.QueryMessage) (*pb.QueryResponseMessage, error) {
	log.Printf("Receive query %s", in.Query)
	return &pb.QueryResponseMessage{Response: "bar"}, nil
}
