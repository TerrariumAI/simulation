package main

import (
	"context"
	"errors"
	"log"

	. "github.com/olamai/proto/simulation"
)

// ----------------
// --- DEV ONLY ---
// ----------------

// Create a new agent and return the new agent's id
func (s *Server) CreateAgent(ctx context.Context, req *CreateAgentRequest) (*Agent, error) {
	// Attempt to spawn the agent
	success, agent := s.world.SpawnAgent(Vec2{req.Agent.X, req.Agent.Y})

	// If unsuccessful, throw an error
	if !success {
		// Throw error if the agent couldn't spawn
		err := errors.New("SpawnAgent(): Agent couldn't spawn in that position")
		return nil, err
	}

	// If succesful, return the agent's id
	return &Agent{Id: agent.id, X: agent.pos.x, Y: agent.pos.y}, nil
}

// Get data for an agent
func (s *Server) GetAgent(ctx context.Context, req *GetAgentRequest) (*Agent, error) {
	e, ok := s.world.entities[req.Agent]
	if ok {
		return &Agent{Id: e.id, X: e.pos.x, Y: e.pos.y}, nil
	} else {
		err := errors.New("GetAgent(): Agent Not Found")
		return nil, err
	}
}

// Create a new agent and return the new agent's id
func (s *Server) DeleteAgent(ctx context.Context, req *DeleteAgentRequest) (*Empty, error) {
	println("WARNING: DeleteAgent is not implemented yet")

	// If succesful, return the agent's id
	return &Empty{}, nil
}

// Perform an action on an agent's behalf
func (s *Server) ActionAgent(ctx context.Context, req *ActionAgentRequest) (*Empty, error) {
	s.world.PerformEntityAction(req.Agent, req.Action.Direction, req.Action.ActionId)

	return &Empty{}, nil
}

// Get an observation for an agent
func (s *Server) GetObservation(ctx context.Context, req *GetObservationRequest) (*Observation, error) {
	// Parse id from message
	id := req.Agent
	observation := s.world.ObserveById(id)

	return observation, nil
}

// --------------------
// --- END DEV ONLY ---
// --------------------

func (s *Server) CreateSpectator(req *CreateSpectatorRequest, stream Simulation_CreateSpectatorServer) error {
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
	spectatorId := req.Spectator.Id
	s.world.AddSpectatorChannel(spectatorId)
	println("Specator Id: ", spectatorId)

	// Listen for updates and send them to the client
	for {
		cellUpdate := <-s.world.spectatorChannels[spectatorId]
		if err := stream.Send(&cellUpdate); err != nil {
			// Break the sending loop
			break
		}
	}

	// Remove the spectator and clean up
	log.Printf("Spectator left...")
	s.world.RemoveSpectatorChannel(spectatorId)

	return nil
}

func (s *Server) SubscribeSpectator(ctx context.Context, req *SubscribeSpectatorRequest) (*Empty, error) {
	// // Get info about the client
	// client, ok := peer.FromContext(ctx)
	// if !ok {
	// 	return nil, errors.New("ERROR: Couldn't get info about peer")
	// }
	// addr := client.Addr.String()

	// Get spectator id
	// For now it is just the ip of the client
	spectatorId := req.Spectator
	region := req.Region

	s.world.SubscribeToRegion(spectatorId, Vec2{region.X, region.Y})
	return &Empty{}, nil
}

// Reset the world
func (s *Server) ResetWorld(ctx context.Context, req *Empty) (*Empty, error) {
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
		s.world.BroadcastCellUpdate(pos, e.class)
	}

	return &Empty{}, nil
}
