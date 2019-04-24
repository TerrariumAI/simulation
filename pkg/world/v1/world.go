package world

import (
	"math/rand"
	"time"

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
	Agents map[int64]*Entity
	// Map from position -> *Entity
	posEntityMap map[vec2.Vec2]*Entity
	// Function callbacks
	onCellUpdate onCellUpdate
}

// NewWorld creates a new world objects
func NewWorld(regionSize int32, onCellUpdate onCellUpdate, shouldSpawnFood bool) World {
	world := World{
		regionSize:   regionSize,
		entities:     map[int64]*Entity{},
		Agents:       map[int64]*Entity{},
		posEntityMap: map[vec2.Vec2]*Entity{},
		onCellUpdate: onCellUpdate,
	}
	if shouldSpawnFood {
		world.startFoodSpawnTimer()
		world.spawnRandomFood()
	}

	return world
}

// Reset resets the world's entities
func (w *World) Reset() {
	w.entities = make(map[int64]*Entity)
	w.posEntityMap = make(map[vec2.Vec2]*Entity)
	w.spawnRandomFood()
}

// GetObservationCellsForPosition gets all observations for a specific position
func (w *World) GetObservationCellsForPosition(pos vec2.Vec2) []string {
	var cells []string
	// TODO - implement this
	for y := pos.Y + 1; y >= pos.Y-1; y-- {
		for x := pos.X - 1; x <= pos.X+1; x++ {
			var posToObserve = vec2.Vec2{X: x, Y: y}
			// Make sure we don't observe ourselves
			if posToObserve == pos {
				continue
			}
			// Add value from cell
			if entity, ok := w.posEntityMap[posToObserve]; ok {
				cells = append(cells, entity.Class)
			} else {
				cells = append(cells, "EMPTY")
			}
		}
	}
	return cells
}

// Checks if a cell is currently occupied
func (w *World) isCellOccupied(pos vec2.Vec2) bool {
	if _, ok := w.posEntityMap[pos]; ok {
		return true
	}
	return false
}

// Spawn random food entities around the world
func (w *World) spawnRandomFood() {
	for i := 0; i < 200; i++ {
		x := int32(rand.Intn(50) - 25)
		y := int32(rand.Intn(50) - 25)
		// Don't put anything at 0,0
		if x == 0 && y == 0 {
			continue
		}
		w.NewFoodEntity(vec2.Vec2{X: x, Y: y})
	}
}

func (w *World) startFoodSpawnTimer() {
	// Creates a ticker and a quit channel, in case we want to stop this
	//  timer in the future. At the moment there is no need for it though
	ticker := time.NewTicker(1 * time.Minute)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				if w.foodCount < minFoodBeforeRespawn {
					w.spawnRandomFood()
				}
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}
