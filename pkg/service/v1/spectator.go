package v1

import (
	"context"
	"errors"
	"log"
	"time"

	v1 "github.com/olamai/simulation/pkg/api/v1"
	"github.com/olamai/simulation/pkg/logger"
	"github.com/olamai/simulation/pkg/vec2/v1"
)

// Remove an agent
func (s *simulationServiceServer) CreateSpectator(req *v1.CreateSpectatorRequest, stream v1.SimulationService_CreateSpectatorServer) error {
	// Get spectator ID from client in the request
	spectatorID := req.Id
	// Lock the data, unlock after spectator is added
	s.m.Lock()
	channel := s.stadium.AddSpectator(spectatorID)
	// Unlock data
	s.m.Unlock()

	// Listen for updates and send them to the client
	for {
		response := <-channel
		if err := stream.Send(&response); err != nil {
			// Break the sending loop
			break
		}
	}

	// Remove the spectator and clean up
	// Lock data until spectator is removed
	s.m.Lock()
	s.stadium.RemoveSpectator(spectatorID)
	// Unlock data
	s.m.Unlock()
	log.Printf("Spectator left...")

	return nil
}

// Get an observation for an agent
func (s *simulationServiceServer) SubscribeSpectatorToRegion(ctx context.Context, req *v1.SubscribeSpectatorToRegionRequest) (*v1.SubscribeSpectatorToRegionResponse, error) {
	// customHeader := ctx.Value("custom-header=1")
	id := req.Id
	region := vec2.Vec2{X: req.Region.X, Y: req.Region.Y}

	// Lock the data while creating the spectator
	s.m.Lock()
	// If the user is already subbed, successful is false
	successful := s.stadium.SubscribeSpectatorToRegion(id, region)
	if !successful {
		s.m.Unlock()
		return &v1.SubscribeSpectatorToRegionResponse{
			Api:        apiVersion,
			Successful: false,
		}, nil
	}
	// Get spectator channel
	exists := s.stadium.DoesSpectatorExist(id)
	// Unlock the data
	s.m.Unlock()

	// If the channel hasn't been created yet, try waiting a couple seconds then trying again
	//  Try this 3 times
	for i := 1; i < 4; i++ {
		if exists {
			break
		}
		logger.Log.Warn("SubscribeSpectatorToRegion(): Spectator channel is nil, sleeping and trying again. Try #" + string(i))
		time.Sleep(2 * time.Second)
		// Lock the data when attempting to read from spect map
		s.m.Lock()
		exists = s.stadium.DoesSpectatorExist(id)
		// Unlock the data
		s.m.Unlock()
	}

	// If after the retrys it still hasn't found a channel throw an error
	if !exists {
		return nil, errors.New("SubscribeSpectatorToRegion(): Couldn't find a spectator by that id")
	}

	// Lock the data while sending the spectator the initial region data
	s.m.Lock()
	defer s.m.Unlock()

	// Send initial region state
	positions := region.GetPositionsInRegion(regionSize)
	for _, pos := range positions {
		if entity := s.world.GetEntityByPos(pos); entity != nil {
			s.stadium.SendCellUpdate(id, pos, entity)
		}
	}

	return &v1.SubscribeSpectatorToRegionResponse{
		Api:        apiVersion,
		Successful: true,
	}, nil
}

func (s *simulationServiceServer) UnsubscribeSpectatorFromRegion(ctx context.Context, req *v1.UnsubscribeSpectatorFromRegionRequest) (*v1.UnsubscribeSpectatorFromRegionResponse, error) {
	// customHeader := ctx.Value("custom-header=1")
	id := req.Id
	region := vec2.Vec2{X: req.Region.X, Y: req.Region.Y}

	// Lock the data while creating the spectator
	s.m.Lock()
	defer s.m.Unlock()
	// Attempt to unsub
	successful := s.stadium.UnsubscribeSpectatorFromRegion(id, region)
	if !successful {
		return &v1.UnsubscribeSpectatorFromRegionResponse{
			Api:        apiVersion,
			Successful: false,
		}, nil
	}

	return &v1.UnsubscribeSpectatorFromRegionResponse{
		Api:        apiVersion,
		Successful: true,
	}, nil
}
