package main

import pb "github.com/olamai/proto"

// Server represents the gRPC server
type Server struct {
	agents        map[int32]*Entity
	chAgentSpawn  chan SpawnAgentWithNewAgentIdChan
	chAgentAction chan pb.AgentActionRequest

	// Observer id to use for next observer
	observerId int32
	// Entity id to use for next entity
	entityId int32

	// Map from observer id to their observation channel
	observerationChannels map[int32]chan EntityUpdate
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
