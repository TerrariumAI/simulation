package main

import (
	"math/rand"

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

func NewWorld() World {
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

func (w *World) MoveEntity(id string, pos Vec2) bool {
	e, ok := w.entities[id]

	// [Start Checks]
	// Make sure the entity exists
	if !ok {
		return false
	}
	// Make sure space is empty
	if _, ok := w.posEntityMatrix[pos]; ok {
		return false
	}
	// [End Checkss]

	// All checks have passed, move the entity
	e.Pos = pos
	w.posEntityMatrix[pos] = e

	return true
}

func (w *World) ObserveByPosition(pos Vec2) []string {
	var observation []string
	// TODO - implement this
	for x := pos.X - 1; x <= pos.X+1; x++ {
		for y := pos.Y + 1; y >= pos.Y-1; y-- {
			var posToObserve = Vec2{x, y}
			// Make sure we don't observe ourselves
			if posToObserve == pos {
				continue
			}
			println("Observing: ", x, y)
			// Add observation from cell
			if entity, ok := w.posEntityMatrix[posToObserve]; ok {
				observation = append(observation, entity.Class)
			} else {
				observation = append(observation, "EMPTY")
			}
		}
	}
	return observation
}

func (w *World) ObserveById(id string) (success bool, observation []string) {
	// If entity exists, return true success and the observation
	if e, ok := w.entities[id]; ok {
		return true, w.ObserveByPosition(e.Pos)
	}
	// Return false for success and empty slice
	return false, make([]string, 0)
}
