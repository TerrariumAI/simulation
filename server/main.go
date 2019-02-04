package main

import (
	"log"
	"net"

	pb "github.com/olamai/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type vec2 struct {
	X int32
	Y int32
}

type Entity struct {
	Id    string
	Class string
	Pos   vec2
}

// Update message for an entity. Can either be an update or a message
//  saying the entity has died or been removed (i.e. food being eaten).
type EntityUpdate struct {
	Action string
	Entity Entity
}

// Create an abstraction around pb.SpawnAgentRequest that has a channel
//  where the new agent's id can be sent back
type SpawnAgentWithNewAgentIdChan struct {
	// The message that was sent to spawn the agent
	msg pb.SpawnAgentRequest
	// Channel to send the result to
	chNewAgentId chan string
}

func main() {
	// Initialize server obj
	var simulationServer = Server{
		agents:                make(map[string]*Entity),
		chAgentSpawn:          make(chan SpawnAgentWithNewAgentIdChan, 3),
		chAgentAction:         make(chan pb.AgentActionRequest, 3),
		observerationChannels: make(map[string]chan EntityUpdate),
	}

	// Start the simulation on another thread
	go simulationServer.startSimulation()

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
