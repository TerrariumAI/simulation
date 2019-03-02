package main

import (
	"math/rand"
	"time"

	. "github.com/olamai/proto/simulation"
)

type World struct {
	// Entity storage
	entities map[string]*Entity
	// Map from position -> *Entity
	posEntityMatrix map[Vec2]*Entity
	// Map from observer id to their observation channel
	spectatorChannels map[string]chan CellUpdate
	// Specators subscribe to regions
	regionSubs map[Vec2][]string
}

const agent_living_energy_cost = 5
const agent_no_energy_health_cost = 10
const region_size = 10

func NewWorld() World {
	// Seed random
	rand.Seed(time.Now().UnixNano())
	// Create world
	w := World{
		entities:          make(map[string]*Entity),
		posEntityMatrix:   make(map[Vec2]*Entity),
		spectatorChannels: make(map[string]chan CellUpdate),
		regionSubs:        make(map[Vec2][]string),
	}
	w.SpawnEntity(Vec2{0, 1}, "FOOD")
	// Spawn food randomly
	for i := 0; i < 50; i++ {
		x := int32(rand.Intn(50) - 25)
		y := int32(rand.Intn(50) - 25)
		// Don't put anything at 0,0
		if x == 0 || y == 0 {
			continue
		}
		w.SpawnEntity(Vec2{x, y}, "FOOD")
	}
	return w
}

// -------------------
// --- Spectation ---
// -------------------
func (w *World) AddSpectatorChannel(id string) string {
	// id := uuid.Must(uuid.NewV4()).String()
	w.spectatorChannels[id] = make(chan CellUpdate, 100)
	return id
}

func (w *World) RemoveSpectatorChannel(id string) {
	// Loop over regions
	for region, spectatorIds := range w.regionSubs {
		// If the user is subscribed to this region, remove their subscription
		for i, spectatorId := range spectatorIds {
			if spectatorId == id {
				w.regionSubs[region] = append(spectatorIds[:i], spectatorIds[i+1:]...)
				break
			}
		}
	}
	delete(w.spectatorChannels, id)
}

func (w *World) BroadcastCellUpdate(pos Vec2, occupant string) {
	// Get region for this position
	region := pos.GetRegion()
	// Get subs for this region
	subs := w.regionSubs[region]
	// Loop over and send to channel
	for _, spectatorId := range subs {
		channel := w.spectatorChannels[spectatorId]
		channel <- CellUpdate{X: pos.x, Y: pos.y, Occupant: occupant}
	}
}

func (w *World) isSpectatorAlreadySubscribedToRegion(spectatorId string, region Vec2) bool {
	// Get subs for this region
	subs := w.regionSubs[region]
	// Loop over and send to channel
	for _, _spectatorId := range subs {
		if _spectatorId == spectatorId {
			return true
		}
	}
	return false
}

func (w *World) SubscribeToRegion(spectatorId string, region Vec2) bool {
	if w.isSpectatorAlreadySubscribedToRegion(spectatorId, region) {
		return false
	}
	// Add spectator id to subscription slice
	w.regionSubs[region] = append(w.regionSubs[region], spectatorId)
	// Get spectator channel
	channel := w.spectatorChannels[spectatorId]
	// Send initial world state
	xs, ys := region.GetPositionsInRegion()
	for _, x := range xs {
		for _, y := range ys {
			pos := Vec2{x, y}
			if entity, ok := w.posEntityMatrix[pos]; ok {
				channel <- CellUpdate{X: pos.x, Y: pos.y, Occupant: entity.class}
			}
		}
	}
	return true
}

// -------------------
// ----- Agents ------
// -------------------
func (w *World) SpawnAgent(pos Vec2) (success bool, entity *Entity) {
	// Check to see if there is already an entity in that position
	// If so, return false and don't spawn
	if _, ok := w.posEntityMatrix[pos]; ok {
		return false, nil
	}

	// Create the entity and add to entities map AND position matrix
	e := NewEntity("AGENT", pos)
	w.entities[e.id] = &e
	w.posEntityMatrix[pos] = &e

	// Send to observation
	w.BroadcastCellUpdate(pos, e.class)

	return true, &e
}

// -------------------
// ---- Entities -----
// -------------------
func (w *World) SpawnEntity(pos Vec2, class string) (success bool, e *Entity) {
	// Check to see if there is already an entity in that position
	// If so, return false and don't spawn
	if _, ok := w.posEntityMatrix[pos]; ok {
		return false, nil
	}

	// Create the entity and add to entities map AND position matrix
	entity := NewEntity(class, pos)
	w.entities[entity.id] = &entity
	w.posEntityMatrix[pos] = &entity

	// Send to observation
	w.BroadcastCellUpdate(pos, class)

	return true, &entity
}

func (w *World) RemoveEntityById(id string) (success bool) {
	e, ok := w.entities[id]
	// Check to see if the entity exists
	// If not, return false
	if !ok {
		return false
	}

	pos := e.pos
	// Remove keys from maps
	delete(w.entities, id)
	delete(w.posEntityMatrix, pos)

	// Send to observation
	w.BroadcastCellUpdate(pos, "EMPTY")

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
	delete(w.posEntityMatrix, e.pos)
	// Send to observation
	w.BroadcastCellUpdate(e.pos, "EMPTY")

	// Move the entity to new position
	e.pos = targetPos
	w.posEntityMatrix[targetPos] = e
	// Send to observation
	w.BroadcastCellUpdate(e.pos, e.class)

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
		return false
	} else {
		if targetEntity.class != "FOOD" {
			return false
		}
	}
	// [End Checks]

	// Remove food entity
	w.RemoveEntityById(targetEntity.id)
	// Add to current entity's energy
	e.energy += 10

	return true
}

func (w *World) PerformEntityAction(id string, direction string, action string) bool {
	var actionSuccess bool

	// Get the entity by id
	e, ok := w.entities[id]
	if !ok {
		return false
	}

	// Don't do anything if you don't have energy left
	if e.energy == 0 {
		e.health -= 10
	}

	// If it's health is 0, remove (kill) the entity and return false
	if e.health <= 0 {
		w.RemoveEntityById(id)
		return false
	}

	// Get the target position from the given direction
	var targetPos Vec2
	switch direction {
	case "UP":
		targetPos = Vec2{e.pos.x, e.pos.y + 1}
	case "DOWN":
		targetPos = Vec2{e.pos.x, e.pos.y - 1}
	case "LEFT":
		targetPos = Vec2{e.pos.x - 1, e.pos.y}
	case "RIGHT":
		targetPos = Vec2{e.pos.x + 1, e.pos.y}
	default: // Direction not correct
		return false
	}

	// Perform the action
	switch action {
	case "MOVE":
		actionSuccess = w.EntityMove(id, targetPos)
	case "CONSUME":
		actionSuccess = w.EntityConsume(id, targetPos)
	}

	// Take off living expense
	e.energy -= agent_living_energy_cost
	if e.energy < 0 {
		e.energy = 0
	}

	// Action not identified
	return actionSuccess
}

func (w *World) GetObservationCellsForPosition(pos Vec2) []string {
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
			if entity, ok := w.posEntityMatrix[posToObserve]; ok {
				cells = append(cells, entity.class)
			} else {
				cells = append(cells, "EMPTY")
			}
		}
	}
	return cells
}

func (w *World) ObserveById(id string) (observation *Observation) {
	// If entity exists, return true success and the observation
	e, ok := w.entities[id]
	if ok {
		cells := w.GetObservationCellsForPosition(e.pos)
		return &Observation{Alive: true, Cells: cells, Energy: e.energy, Health: e.health}
	} else {
		return &Observation{Alive: false, Cells: []string{}, Energy: 0, Health: 0}
	}
}
