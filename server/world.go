package main

import (
	"math/rand"
	"time"

	pb "github.com/olamai/proto"
	uuid "github.com/satori/go.uuid"
)

type World struct {
	// Entity storage
	entities map[string]*Entity
	// Map from position -> *Entity
	posEntityMatrix map[Vec2]*Entity
	// Map from observer id to their observation channel
	observerationChannels map[string]chan pb.CellUpdate
}

const agent_living_energy_cost = 5

func NewWorld() World {
	// Seed random
	rand.Seed(time.Now().UnixNano())
	// Create world
	w := World{
		entities:              make(map[string]*Entity),
		posEntityMatrix:       make(map[Vec2]*Entity),
		observerationChannels: make(map[string]chan pb.CellUpdate),
	}
	w.SpawnEntity(Vec2{0, 1}, "FOOD")
	// Spawn food randomly
	for i := 0; i < 3000; i++ {
		x := int32(rand.Intn(200) - 100)
		y := int32(rand.Intn(200) - 100)
		w.SpawnEntity(Vec2{x, y}, "FOOD")
	}
	return w
}

// -------------------
// --- Observation ---
// -------------------
func (w *World) AddObservationChannel() string {
	id := uuid.Must(uuid.NewV4()).String()
	w.observerationChannels[id] = make(chan pb.CellUpdate)
	return id
}

func (w *World) RemoveObservationChannel(id string) {
	delete(w.observerationChannels, id)
}

func (w *World) BroadcastCellUpdate(pos Vec2, occupant string) {
	for _, channel := range w.observerationChannels {
		channel <- pb.CellUpdate{X: pos.X, Y: pos.Y, Occupant: occupant}
	}
}

// -------------------
// ----- Agents ------
// -------------------
func (w *World) SpawnAgent(pos Vec2) (success bool, id string) {
	// Check to see if there is already an entity in that position
	// If so, return false and don't spawn
	if _, ok := w.posEntityMatrix[pos]; ok {
		return false, ""
	}

	// Create the entity and add to entities map AND position matrix
	e := NewEntity("AGENT", pos)
	w.entities[e.Id] = &e
	w.posEntityMatrix[pos] = &e

	return true, e.Id
}

// -------------------
// ---- Entities -----
// -------------------
func (w *World) SpawnEntity(pos Vec2, class string) (success bool, id string) {
	// Check to see if there is already an entity in that position
	// If so, return false and don't spawn
	if _, ok := w.posEntityMatrix[pos]; ok {
		return false, ""
	}

	// Create the entity and add to entities map AND position matrix
	e := NewEntity(class, pos)
	w.entities[e.Id] = &e
	w.posEntityMatrix[pos] = &e

	return true, e.Id
}

func (w *World) RemoveEntityById(id string) (success bool) {
	e, ok := w.entities[id]
	// Check to see if the entity exists
	// If not, return false
	if !ok {
		return false
	}

	pos := e.Pos
	// Make sure nothing points to the Entity anymore so it can be thrown out
	w.entities[id] = nil
	w.posEntityMatrix[pos] = nil
	// Remove keys from maps
	delete(w.entities, id)
	delete(w.posEntityMatrix, pos)

	return true
}

func (w *World) EntityMove(id string, targetPos Vec2) bool {
	e, ok := w.entities[id]

	// [Start Checks]
	// Make sure the entity exists
	if !ok {
		return false
	}
	// Make sure space is empty
	if _, ok := w.posEntityMatrix[targetPos]; ok {
		return false
	}
	// [End Checks]

	// Remove entity from current position
	delete(w.posEntityMatrix, e.Pos)
	// Move the entity to new position
	e.Pos = targetPos
	w.posEntityMatrix[targetPos] = e

	return true
}

func (w *World) EntityConsume(id string, targetPos Vec2) bool {
	e, ok := w.entities[id]

	// [Start Checks]
	// Make sure the entity exists
	if !ok {
		return false
	}
	// Make sure space is empty
	targetEntity, ok := w.posEntityMatrix[targetPos]
	if !ok {
		if e.Class != "FOOD" {
			return false
		}
	}
	// [End Checks]

	// Remove food entity
	w.RemoveEntityById(targetEntity.Id)
	// Add to current entity's energy
	e.Energy += 10

	return true
}

func (w *World) PerformEntityAction(id string, direction string, action string) bool {
	var actionSuccess bool

	e, ok := w.entities[id]
	if !ok {
		return false
	}

	var targetPos Vec2
	switch direction {
	case "UP":
		targetPos = Vec2{e.Pos.X, e.Pos.Y + 1}
	case "DOWN":
		targetPos = Vec2{e.Pos.X, e.Pos.Y - 1}
	case "LEFT":
		targetPos = Vec2{e.Pos.X - 1, e.Pos.Y}
	case "RIGHT":
		targetPos = Vec2{e.Pos.X + 1, e.Pos.Y}
	default: // Direction not correct
		return false
	}

	switch action {
	case "MOVE":
		actionSuccess = w.EntityMove(id, targetPos)
	case "CONSUME":
		actionSuccess = w.EntityConsume(id, targetPos)
	default:
		return false
	}

	// Take off living expense
	e.Energy -= agent_living_energy_cost

	// Action not identified
	return actionSuccess
}

func (w *World) GetObservationCellsForPosition(pos Vec2) []string {
	var cells []string
	// TODO - implement this
	for y := pos.Y + 1; y >= pos.Y-1; y-- {
		for x := pos.X - 1; x <= pos.X+1; x++ {
			var posToObserve = Vec2{x, y}
			// Make sure we don't observe ourselves
			if posToObserve == pos {
				continue
			}
			// Add value from cell
			if entity, ok := w.posEntityMatrix[posToObserve]; ok {
				cells = append(cells, entity.Class)
			} else {
				cells = append(cells, "EMPTY")
			}
		}
	}
	return cells
}

func (w *World) ObserveById(id string) (success bool, observation *pb.AgentObservationResult) {
	var cells []string
	// If entity exists, return true success and the observation
	e, ok := w.entities[id]
	if ok {
		cells = w.GetObservationCellsForPosition(e.Pos)
	} else {
		return false, nil
	}
	return true, &pb.AgentObservationResult{Cells: cells, Energy: e.Energy}
}
