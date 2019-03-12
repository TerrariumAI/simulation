package v1

import (
	"crypto/rand"
	"fmt"
	"io"
)

const (
	regionSize = 10
)

// Vec2 - Simple struct for holding positions
type Vec2 struct {
	x int32
	y int32
}

// GetRegion - Returns the region that a position is in
func (v *Vec2) GetRegion() Vec2 {
	x := v.x
	y := v.y
	var signX int32 = 1
	var signY int32 = 1
	if x < 0 {
		signX = -1
	}
	if y < 0 {
		signY = -1
	}
	return Vec2{x/10 + signX, y/10 + signY}
}

// GetPositionsInRegion - Returns all positions that are in a specfic region
func (v *Vec2) GetPositionsInRegion() ([]int32, []int32) {
	xs := []int32{}
	ys := []int32{}
	var signX int32 = 1
	var signY int32 = 1
	if v.x < 0 {
		signX = -1
	}
	if v.y < 0 {
		signY = -1
	}
	startX := (v.x - signX) * regionSize
	startY := (v.y - signY) * regionSize
	endX := v.x * regionSize
	endY := v.y * regionSize
	if signX > 0 {
		for x := startX; x < endX; x++ {
			xs = append(xs, x)
		}
	} else {
		for x := startX; x > endX; x-- {
			xs = append(xs, x)
		}
	}
	if signY > 0 {
		for y := startY; y < endY; y++ {
			ys = append(ys, y)
		}
	} else {
		for y := startY; y > endY; y-- {
			ys = append(ys, y)
		}
	}

	return xs, ys
}

// newUUID generates a random UUID according to RFC 4122
func newUUID() (string, error) {
	uuid := make([]byte, 16)
	n, err := io.ReadFull(rand.Reader, uuid)
	if n != len(uuid) || err != nil {
		return "", err
	}
	// variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80
	// version 4 (pseudo-random); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:]), nil
}

// ---------------------
// Simulation utils
// ---------------------

// Get all observations for a specific position
func (s *simulationServiceServer) GetObservationCellsForPosition(pos Vec2) []string {
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
