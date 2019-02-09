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

	// Test Spawn
	spawnResp, err := c.SpawnAgent(context.Background(), &pb.SpawnAgentRequest{X: 0, Y: 0})
	if err != nil {
		t.Errorf("error when calling SpawnAgent: %s", err)
	}
	println("Spawned new agent with ID: ", spawnResp.Id)
	agentId := spawnResp.Id

	// Test Observation
	obsvResp, err := c.AgentObservation(context.Background(), &pb.AgentObservationRequest{Id: agentId})
	if err != nil {
		t.Errorf("error when calling SpawnAgent: %s", err)
	}
	println("Agent Observation: ", obsvResp.Cells[0])

	// // Test Action
	// actionResp, err := c.AgentAction(context.Background(), &pb.AgentActionRequest{Id: agentId, Action: "UP"})
	// if err != nil {
	// 	t.Errorf("error when calling SpawnAgent: %s", err)
	// }
	// println("Agent Observation: ", actionResp.Successful)
}
