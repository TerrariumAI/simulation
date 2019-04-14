package v1

import (
	cryptoRand "crypto/rand"
	"errors"
	"fmt"
	"io"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/olamai/simulation/pkg/vec2/v1"
	"github.com/olamai/simulation/pkg/world/v1"
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
func getTargetPosFromDirectionAndAgent(dir string, agent *world.Entity) (vec2.Vec2, error) {
	switch dir {
	case "UP":
		return vec2.Vec2{X: agent.Pos.X, Y: agent.Pos.Y + 1}, nil
	case "DOWN":
		return vec2.Vec2{X: agent.Pos.X, Y: agent.Pos.Y - 1}, nil
	case "LEFT":
		return vec2.Vec2{X: agent.Pos.X - 1, Y: agent.Pos.Y}, nil
	case "RIGHT":
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
