package main

import (
	"context"
	"errors"
	"log"

	pb "github.com/olamai/proto"
)

// Create a new agent and return the new agent's id
func (s *Server) SpawnAgent(ctx context.Context, in *pb.SpawnAgentRequest) (*pb.SpawnAgentResult, error) {
	log.Printf("Receive spawn Entity message %s", in.X)
	log.Printf("Receive spawn Entity message %s", in.Y)

	chNewAgentId := make(chan int32)
	s.chSpawnAgent <- SpawnAgentWithNewAgentIdChan{msg: *in, chNewAgentId: chNewAgentId}
	// Wait for the resulting new agent id
	newAgentId := <-chNewAgentId
	return &pb.SpawnAgentResult{Id: newAgentId}, nil
}

func (s *Server) AgentObservation(ctx context.Context, in *pb.AgentObservationRequest) (*pb.AgentObservationResult, error) {
	log.Printf("AgentObservation()")

	// Parse id from message
	id := in.Id

	if _, ok := s.agents[id]; ok {
		var entities []*pb.Entity
		// Loop over agents and add to entities
		// TODO - only return agent's close to this agent rather than all of them
		for id, otherAgent := range s.agents {
			// Add agent's data to a PB message
			e := &pb.Entity{
				Id:    id,
				Class: otherAgent.Class,
				X:     otherAgent.Pos.X,
				Y:     otherAgent.Pos.Y,
			}
			entities = append(entities, e)
		}

		// TODO - loop over other entities such as food and also add

		// Return the observation data
		return &pb.AgentObservationResult{
			Entities: entities,
		}, nil
	} else {
		// Throw error if the agent doesn't exist
		err := errors.New("AgentObservation(): Agent with that Id doesn't exist")
		return nil, err
	}
}

func (s *Server) Spectate(ctx context.Context, obsvRequest *pb.SpectateRequest, stream pb.Simulation_SpectateServer) error {
	observerId := s.GetObserverId()
	// Create observation channel for this observer
	s.observerationChannels[observerId] = make(chan EntityUpdate)
	// Listen for updates and send them to the client
	for {
		update := <-s.observerationChannels[observerId]
		updateMessage := pb.EntityUpdate{
			Action: update.Action,
			Entity: &pb.Entity{
				Id:    update.Entity.Id,
				Class: update.Entity.Class,
				X:     update.Entity.Pos.X,
				Y:     update.Entity.Pos.Y,
			},
		}
		stream.Send(&updateMessage)
	}

	// Remove the observation channel
	delete(s.observerationChannels, observerId)
	return nil
}
