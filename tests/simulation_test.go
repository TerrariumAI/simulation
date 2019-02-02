package tests

import (
	"log"

	"testing"

	pb "github.com/olamai/proto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func TestSimulation(t *testing.T) {
	var conn *grpc.ClientConn

	// Initiate a connection with the server
	conn, err := grpc.Dial(":9090", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %s", err)
	}
	defer conn.Close()

	c := pb.NewSimulationClient(conn)

	// Test Execution
	resp, err := c.SpawnAgent(context.Background(), &pb.SpawnAgentRequest{X: 0, Y: 0})
	if err != nil {
		t.Errorf("error when calling SpawnAgent: %s", err)
	}
	log.Println(resp)
}
