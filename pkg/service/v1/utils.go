package v1

import (
	cryptoRand "crypto/rand"
	"errors"
	"fmt"
	"io"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	v1 "github.com/terrariumai/simulation/pkg/api/v1"
	"github.com/terrariumai/simulation/pkg/vec2/v1"
	"github.com/terrariumai/simulation/pkg/world/v1"
)

// newUUID generates a random UUID according to RFC 4122
func newUUID() (string, error) {
	uuid := make([]byte, 16)
	n, err := io.ReadFull(cryptoRand.Reader, uuid)
	if n != len(uuid) || err != nil {
		return "", err
	}
	// variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80
	// version 4 (pseudo-random); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:]), nil
}

// Given a direction and an agent, return the target position
//  i.e. an agent at (0,0) and direction "UP" returns (0, 1)
func getTargetPosFromDirectionAndAgent(dir uint32, agent *world.Entity) (vec2.Vec2, error) {
	switch dir {
	case 0: // UP
		return vec2.Vec2{X: agent.Pos.X, Y: agent.Pos.Y + 1}, nil
	case 1: // DOWN
		return vec2.Vec2{X: agent.Pos.X, Y: agent.Pos.Y - 1}, nil
	case 2: // LEFT
		return vec2.Vec2{X: agent.Pos.X - 1, Y: agent.Pos.Y}, nil
	case 3: // RIGHT
		return vec2.Vec2{X: agent.Pos.X + 1, Y: agent.Pos.Y}, nil
	default: // Direction not correct
		return vec2.Vec2{}, errors.New("GetTargetPosFromDirectionAndAgent(): Invalid Action.Direction")
	}
}

// checkAPI checks if the API version requested by client is supported by server
func (s *simulationServiceServer) checkAPI(api string) error {
	// API version is "" means use current version of the service
	if len(api) > 0 {
		if apiVersion != api {
			return status.Errorf(codes.Unimplemented,
				"unsupported API version: service implements API version '%s', but asked for '%s'", apiVersion, api)
		}
	}
	return nil
}

// Performs a single steo which goes through every agent and
//  if it has a remote model, sends out an observation.
//  If not, it simply applies living cost to the agent.
func (s *simulationServiceServer) stepWorldOnce() {
	for _, e := range s.world.Agents {
		// Get the RM array for the owner of this agent
		ownerUID := e.OwnerUID
		userRMs := s.remoteModelMap[ownerUID]
		for _, RM := range userRMs {
			// Only use the model connected to this agent
			if RM.name != e.ModelName {
				continue
			}
			// Get the channel for the RM
			RMChannel := RM.channel
			// Get and send the observation to the RM
			cells := s.world.GetObservationCellsForPosition(e.Pos)
			RMChannel <- v1.Observation{
				IsAlive: true,
				Entity: &v1.Entity{
					Id:    e.ID,
					Class: e.Class,
					Pos: &v1.Vec2{
						X: e.Pos.X,
						Y: e.Pos.Y,
					},
					Energy:    e.Energy,
					Health:    e.Health,
					OwnerUID:  e.OwnerUID,
					ModelName: e.ModelName,
				},
				Cells: cells,
			}
		}
		// Cost of living, eg. remove energy/health
		s.world.EntityLivingCostUpdate(e)
	}
}

// Steps over every agent, then sends an action request to it's RM
func (s *simulationServiceServer) stepWorldContinuous() {
	for {
		// Lock the data
		s.m.Lock()
		s.stepWorldOnce()
		// Unlock the data
		s.m.Unlock()
		// Sleep
		if s.env != "training" {
			time.Sleep((1000 / fps) * time.Millisecond)
		}
	}
}
