package main

import (
	"context"
	"errors"
	"log"

	. "github.com/olamai/proto/simulation"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

// --- DEV ONLY ---
// Create a new agent and return the new agent's id
func (s *Server) SpawnAgent(ctx context.Context, in *SpawnAgentRequest) (*SpawnAgentResult, error) {
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
	return &SpawnAgentResult{Id: id}, nil
}

// --- DEV ONLY ---
func (s *Server) AgentObservation(ctx context.Context, in *AgentObservationRequest) (*AgentObservationResult, error) {
	if s.env == "prod" {
		return nil, errors.New("ERROR: AgentObservation not allowed on production server")
	}

	// Parse id from message
	id := in.Id
	observation := s.world.ObserveById(id)

	return observation, nil

	// if _, ok := s.agentPositions; ok {
	// 	var entities []*Entity
	// 	// Loop over agents and add to entities
	// 	// TODO - only return agent's close to this agent rather than all of them
	// 	for id, otherAgent := range s.agents {
	// 		// Add agent's data to a PB message
	// 		entities = append(entities, otherAgent)
	// 	}

	// 	// TODO - loop over other entities such as food and also add

	// 	// Return the observation data
	// 	return &AgentObservationResult{
	// 		Entities: entities,
	// 	}, nil
	// } else {
	// 	// Throw error if the agent doesn't exist
	// 	err := errors.New("AgentObservation(): Agent with that Id doesn't exist")
	// 	return nil, err
	// }
}

// --- DEV ONLY ---
func (s *Server) AgentAction(ctx context.Context, req *AgentActionRequest) (*AgentActionResult, error) {
	if s.env == "prod" {
		return nil, errors.New("ERROR: Action not allowed on this server")
	}

	success := s.world.PerformEntityAction(req.Id, req.Direction, req.Action)

	return &AgentActionResult{Successful: success}, nil
}

func (s *Server) ResetWorld(ctx context.Context, req *ResetWorldRequest) (*ResetWorldResult, error) {
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

	return &ResetWorldResult{}, nil
}

// --- END DEV ONLY ---
func connectionOnState(ctx context.Context, conn *grpc.ClientConn, states ...connectivity.State) <-chan struct{} {
	done := make(chan struct{})

	go func() {
		// any return from this func will close the channel
		defer close(done)

		// continue checking for state change
		// until one of break states is found
		for {
			change := conn.WaitForStateChange(ctx, conn.GetState())
			if !change {
				// ctx is done, return
				// something upstream is cancelling
				return
			}

			currentState := conn.GetState()

			for _, s := range states {
				if currentState == s {
					// matches one of the states passed
					// return, closing the done channel
					return
				}
			}
		}
	}()

	return done
}

func (s *Server) Spectate(req *SpectateRequest, stream Simulation_SpectateServer) error {
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

func (s *Server) SubscribeToRegion(ctx context.Context, req *SubscribeToRegionRequest) (*SubscribeToRegionResult, error) {
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
	return &SubscribeToRegionResult{}, nil
}
