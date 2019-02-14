package main

import (
	"context"
	"errors"
	"log"

	pb "github.com/olamai/proto"
)

// --- DEV ONLY ---
// Create a new agent and return the new agent's id
func (s *Server) SpawnAgent(ctx context.Context, in *pb.SpawnAgentRequest) (*pb.SpawnAgentResult, error) {
	log.Printf("SpawnAgent(): %s", in.X, in.Y)
	if s.env == "prod" {
		return nil, errors.New("ERROR: SpawnAgent not allowed on production server")
	}

	// Attempt to spawn the agent
	success, id := s.world.SpawnAgent(Vec2{in.X, in.Y})

	// If unsuccessful, throw an error
	if !success {
		// Throw error if the agent couldn't spawn
		err := errors.New("SpawnAgent(): Agent couldn't spawn in that position")
		return nil, err
	}

	// If succesful, return the agent's id
	return &pb.SpawnAgentResult{Id: id}, nil
}

// --- DEV ONLY ---
func (s *Server) AgentObservation(ctx context.Context, in *pb.AgentObservationRequest) (*pb.AgentObservationResult, error) {
	log.Printf("AgentObservation()")
	if s.env == "prod" {
		return nil, errors.New("ERROR: AgentObservation not allowed on production server")
	}

	// Parse id from message
	id := in.Id
	success, observation := s.world.ObserveById(id)

	if !success {
		err := errors.New("AgentObservation(): Something went wrong while trying to get that agent's observation data")
		return nil, err
	}

	return observation, nil

	// if _, ok := s.agentPositions; ok {
	// 	var entities []*pb.Entity
	// 	// Loop over agents and add to entities
	// 	// TODO - only return agent's close to this agent rather than all of them
	// 	for id, otherAgent := range s.agents {
	// 		// Add agent's data to a PB message
	// 		entities = append(entities, otherAgent)
	// 	}

	// 	// TODO - loop over other entities such as food and also add

	// 	// Return the observation data
	// 	return &pb.AgentObservationResult{
	// 		Entities: entities,
	// 	}, nil
	// } else {
	// 	// Throw error if the agent doesn't exist
	// 	err := errors.New("AgentObservation(): Agent with that Id doesn't exist")
	// 	return nil, err
	// }
}

// --- DEV ONLY ---
func (s *Server) AgentAction(ctx context.Context, actionReq *pb.AgentActionRequest) (*pb.AgentActionResult, error) {
	log.Printf("AgentAction()")
	if s.env == "prod" {
		return nil, errors.New("ERROR: Action not allowed on this server")
	}

	success := s.world.PerformEntityAction(actionReq.Id, actionReq.Direction, actionReq.Action)

	return &pb.AgentActionResult{Successful: success}, nil
}

func (s *Server) Spectate(req *pb.SpectateRequest, stream pb.Simulation_SpectateServer) error {
	log.Printf("Spectate()")
	log.Printf("Spectator joined...")
	id := s.world.AddObservationChannel()
	// Send initial world state
	for pos, entity := range s.world.posEntityMatrix {
		stream.Send(&pb.CellUpdate{X: pos.X, Y: pos.Y, Occupant: entity.Class})
	}

	// Listen for updates and send them to the client
	for {
		cellUpdate := <-s.world.observerationChannels[id]
		stream.Send(&cellUpdate)
	}

	log.Printf("Spectator left...")
	// Remove the observation channel
	s.world.RemoveObservationChannel(id)
	return nil
}
