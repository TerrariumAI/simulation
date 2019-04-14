package world

import (
	"github.com/olamai/simulation/pkg/vec2/v1"
)

const (
	minFoodBeforeRespawn = 200
)

type onCellUpdate func(vec2.Vec2, *Entity)

// World holds and manages the environment, including holding entities and
//   performing necessary world
type World struct {
	nextEntityID int64
	foodCount    int
	regionSize   int32
	// Map of all entities
	entities map[int64]*Entity
	// Map to keep track of agents
	agents map[int64]*Entity
	// Map from position -> *Entity
	posEntityMap map[vec2.Vec2]*Entity
	// Function callbacks
	onCellUpdate onCellUpdate
}

// NewWorld creates a new world objects
func NewWorld(regionSize int32, onCellUpdate onCellUpdate) World {
	return World{
		regionSize:   regionSize,
		entities:     map[int64]*Entity{},
		agents:       map[int64]*Entity{},
		posEntityMap: map[vec2.Vec2]*Entity{},
		onCellUpdate: onCellUpdate,
	}
}

// Checks if a cell is currently occupied
func (w *World) isCellOccupied(pos vec2.Vec2) bool {
	if _, ok := w.posEntityMap[pos]; ok {
		return true
	}
	return false
}
