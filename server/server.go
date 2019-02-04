package main

import pb "github.com/olamai/proto"

// Server represents the gRPC server
type Server struct {
	agents        map[string]*Entity
	chAgentSpawn  chan SpawnAgentWithNewAgentIdChan
	chAgentAction chan pb.AgentActionRequest

	// Map from observer id to their observation channel
	observerationChannels map[string]chan EntityUpdate
}
