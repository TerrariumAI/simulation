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
	if s.env == "prod" {
		return nil, errors.New("ERROR: AgentObservation not allowed on production server")
	}

	// Parse id from message
	id := in.Id
	observation := s.world.ObserveById(id)

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
func (s *Server) AgentAction(ctx context.Context, req *pb.AgentActionRequest) (*pb.AgentActionResult, error) {
	if s.env == "prod" {
		return nil, errors.New("ERROR: Action not allowed on this server")
	}

	success := s.world.PerformEntityAction(req.Id, req.Direction, req.Action)

	return &pb.AgentActionResult{Successful: success}, nil
}

func (s *Server) ResetWorld(ctx context.Context, req *pb.ResetWorldRequest) (*pb.ResetWorldResult, error) {
	// reset the world, preserving spectator channels
	spectatorChans := s.world.spectatorChannels
	regionSubs := s.world.regionSubs
	s.world = NewWorld()
	s.world.spectatorChannels = spectatorChans
	s.world.regionSubs = regionSubs

	// Broadcast a reset message
	s.world.BroadcastCellUpdate(Vec2{0, 0}, "WORLD_RESET")

	// Broadcast all the new cells
	for pos, e := range s.world.posEntityMatrix {
		s.world.BroadcastCellUpdate(pos, e.Class)
	}

	return &pb.ResetWorldResult{}, nil
}

// --- END DEV ONLY ---

func (s *Server) Spectate(req *pb.SpectateRequest, stream pb.Simulation_SpectateServer) error {
	log.Printf("Spectate()")
	log.Printf("Spectator joined...")
	// // Get info about the client
	// client, ok := peer.FromContext(stream.Context())
	// if !ok {
	// 	return errors.New("ERROR: Couldn't get info about peer")
	// }
	// addr := client.Addr.String()

	// Create a spectator id
	// For now it is just the ip of the client
	spectatorId := req.Id
	s.world.AddSpectatorChannel(spectatorId)
	println("Specator Id: ", spectatorId)

	// Listen for updates and send them to the client
	for {
		cellUpdate := <-s.world.spectatorChannels[spectatorId]
		stream.Send(&cellUpdate)
	}

	log.Printf("Spectator left...")
	// Remove the spectator channel
	s.world.RemoveSpectatorChannel(spectatorId)
	return nil
}

func (s *Server) SubscribeToRegion(ctx context.Context, req *pb.SubscribeToRegionRequest) (*pb.SubscribeToRegionResult, error) {
	// // Get info about the client
	// client, ok := peer.FromContext(ctx)
	// if !ok {
	// 	return nil, errors.New("ERROR: Couldn't get info about peer")
	// }
	// addr := client.Addr.String()

	// Get spectator id
	// For now it is just the ip of the client
	spectatorId := req.Id

	s.world.SubscribeToRegion(spectatorId, Vec2{req.X, req.Y})
	return &pb.SubscribeToRegionResult{}, nil
}
