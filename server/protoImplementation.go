package main

import (
	"context"
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

func (s *Server) Spectate(obsvRequest *pb.SpectateRequest, stream pb.Simulation_SpectateServer) error {
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
