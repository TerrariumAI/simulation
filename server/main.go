package main

import (
	"log"
	"net"

	pb "github.com/olamai/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// Update message for an entity. Can either be an update or a message
//  saying the entity has died or been removed (i.e. food being eaten).
type CellUpdate struct {
	X        int32
	Y        int32
	Occupant string
}

// Create an abstraction around pb.SpawnAgentRequest that has a channel
//  where the new agent's id can be sent back
type SpawnAgentWithNewAgentIdChan struct {
	// The message that was sent to spawn the agent
	msg pb.SpawnAgentRequest
	// Channel to send the result to
	chNewAgentId chan int32
}

func main() {
	// Initialize server obj
	var simulationServer = Server{
		agents:                make(map[int32]*Entity),
		chAgentSpawn:          make(chan SpawnAgentWithNewAgentIdChan, 3),
		chAgentAction:         make(chan pb.AgentActionRequest, 3),
		observerationChannels: make(map[int32]chan EntityUpdate),
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

// TEMPORARY UNTIL OBSV SERVICE IS DEVELOPED
// Returns the next observer's id and increases by 1
func (s *Server) GetObserverId() (id int32) {
	s.observerId += 1
	return s.observerId
}

// Returns the next entity's id and increases by 1
func (s *Server) GetEntityId() (id int32) {
	s.entityId += 1
	return s.entityId
}
