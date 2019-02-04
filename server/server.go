package main

import (
	pb "github.com/olamai/proto"
	uuid "github.com/satori/go.uuid"
)

type vec2 struct {
	X int32
	Y int32
}

type Entity struct {
	Id    int32
	Class string
	Pos   vec2
}

// Server represents the gRPC server
type Server struct {
	agents map[string]*Entity
	// Map x, y -> entity for position map
	entities map[vec2]*Entity
	// Channels
	chAgentSpawn  chan SpawnAgentWithNewAgentIdChan
	chAgentAction chan pb.AgentActionRequest

	// Observer id to use for next observer
	observerId int32
	// Entity id to use for next entity
	entityId int32

	// Map from observer id to their observation channel
	observerationChannels map[int32]chan EntityUpdate
}

// func (s *Server) TeleportEntity(id string, x int32, y int32) bool {
// 	// If something else is already in the position, return false
// 	if entityInPosition, ok := s.gameMap[vec2{x, y}]; ok {
// 		return false
// 	}
// 	// If nothing is there, move
// 	// First, make sure the agent exists
// 	if entity, ok := s.entities[id]; ok {
// 		s.posMap(vec2{x, y}) = agent
// 		return true
// 	} else {
// 		return false
// 	}
// }

func (s *Server) SpawnEntity(class string, x int32, y int32) (id string) {
	// Create the entity for the agent
	id, _ := uuid.NewV4()
	stringId := id.String()
	pos := vec2{x, y}
	e := pb.Entity{Id: stringId, Class: class, Pos: pos}
	s.entities[vec2{x, y}] = &e

	return &e
}
