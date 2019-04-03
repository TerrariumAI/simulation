package v1

import (
	cryptoRand "crypto/rand"
	"errors"
	"fmt"
	"io"
	"math/rand"
)

const (
	regionSize = 16
)

// Vec2 - Simple struct for holding positions
type Vec2 struct {
	x int32
	y int32
}

// GetRegion - Returns the region that a position is in
func (v *Vec2) getRegion() Vec2 {
	x := v.x
	y := v.y
	if x < 0 {
		x -= regionSize
	}
	if y < 0 {
		y -= regionSize
	}
	return Vec2{x / regionSize, y / regionSize}
}

// GetPositionsInRegion - Returns all positions that are in a specfic region
func (v *Vec2) getPositionsInRegion() []Vec2 {
	positions := []Vec2{}
	for x := v.x * regionSize; x < v.x*regionSize+regionSize; x++ {
		for y := v.y * regionSize; y < v.y*regionSize+regionSize; y++ {
			positions = append(positions, Vec2{x, y})
		}
	}
	return positions
}

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
func getTargetPosFromDirectionAndAgent(dir string, agent *Entity) (Vec2, error) {
	switch dir {
	case "UP":
		return Vec2{agent.pos.x, agent.pos.y + 1}, nil
	case "DOWN":
		return Vec2{agent.pos.x, agent.pos.y - 1}, nil
	case "LEFT":
		return Vec2{agent.pos.x - 1, agent.pos.y}, nil
	case "RIGHT":
		return Vec2{agent.pos.x + 1, agent.pos.y}, nil
	default: // Direction not correct
		return Vec2{}, errors.New("GetTargetPosFromDirectionAndAgent(): Invalid Action.Direction")
	}
}

// ---------------------
// Simulation utils
// ---------------------

// Get all observations for a specific position
func (s *simulationServiceServer) getObservationCellsForPosition(pos Vec2) []string {
	var cells []string
	// TODO - implement this
	for y := pos.y + 1; y >= pos.y-1; y-- {
		for x := pos.x - 1; x <= pos.x+1; x++ {
			var posToObserve = Vec2{x, y}
			// Make sure we don't observe ourselves
			if posToObserve == pos {
				continue
			}
			// Add value from cell
			if entity, ok := s.posEntityMap[posToObserve]; ok {
				cells = append(cells, entity.class)
			} else {
				cells = append(cells, "EMPTY")
			}
		}
	}
	return cells
}

func (s *simulationServiceServer) spawnRandomFood() {
	for i := 0; i < 200; i++ {
		x := int32(rand.Intn(50) - 25)
		y := int32(rand.Intn(50) - 25)
		// Don't put anything at 0,0
		if x == 0 && y == 0 {
			continue
		}
		s.newEntity("FOOD", "", "", Vec2{x, y})
	}
}
