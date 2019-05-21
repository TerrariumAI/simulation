package world

import (
	"errors"

	"github.com/terrariumai/simulation/pkg/vec2/v1"
)

const (
	initialEnergy            = 100
	initialHealth            = 100
	entityLivingEnergyCost   = 2
	entityMoveEnergyCost     = 4
	entityConsumeEnergyGain  = 10
	entityNoEnergyHealthCost = 10
)

// Entity - data for entities that exist in cells
type Entity struct {
	ID        int64
	Class     string
	Pos       vec2.Vec2
	Energy    int32
	Health    int32
	OwnerUID  string
	ModelName string
}

// NewEntity creates a new entity and add it to the simulation
func (w *World) newEntity(class string, ownerUID string, modelName string, pos vec2.Vec2) (*Entity, error) {
	// Make sure the cell is empty
	if w.isCellOccupied(pos) {
		err := errors.New("newEntity(): Cell is already occupied")
		return nil, err
	}

	// Create the entity
	id := w.nextEntityID
	w.nextEntityID++
	e := Entity{id, class, pos, initialEnergy, initialHealth, ownerUID, modelName}
	w.entities[id] = &e
	w.posEntityMap[pos] = &e

	// Add to agents map if it is an agent
	if class == "AGENT" {
		w.Agents[id] = &e
	}

	// Broadcast update
	w.onCellUpdate(e.Pos, &e)

	return &e, nil
}

// NewFoodEntity spawns a new food entity at the specified location
func (w *World) NewFoodEntity(pos vec2.Vec2) (*Entity, error) {
	// Add to foodCount stat
	w.foodCount++
	// Create the entity
	return w.newEntity("FOOD", "", "", pos)
}

// NewAgentEntity creates a new agent. Agents are just an abstraction of an entity, essentially
//  just a group of entities.
func (w *World) NewAgentEntity(ownerUID string, modelName string, pos vec2.Vec2) (*Entity, error) {
	return w.newEntity("AGENT", ownerUID, modelName, pos)
}

// DeleteEntity deletes an entity by Id
func (w *World) DeleteEntity(id int64) bool {
	// Get the entitiy
	e, ok := w.entities[id]
	// Return false if an entitiy by that id doesn't exist
	if !ok {
		return false
	}

	// Handle class specific removals
	if e.Class == "AGENT" {
		// Remove from agents map if it is an agent
		delete(w.Agents, e.ID)
	} else if e.Class == "FOOD" {
		// Subtract from foodCount
		w.foodCount--
	}

	// Remove the entity
	delete(w.entities, e.ID)
	delete(w.posEntityMap, e.Pos)

	// Broadcast update
	w.onCellUpdate(e.Pos, nil)

	return true
}

// GetEntity returns the entity with the given id
func (w *World) GetEntity(id int64) *Entity {
	entity, ok := w.entities[id]
	if !ok {
		return nil
	}
	return entity
}

// GetEntityByPos returns the entity with the given position
func (w *World) GetEntityByPos(pos vec2.Vec2) *Entity {
	entity, ok := w.posEntityMap[pos]
	if !ok {
		return nil
	}
	return entity
}

// EntityMove moves an entity, returns if the move was successful
func (w *World) EntityMove(id int64, targetPos vec2.Vec2) bool {
	// Get the entity by id
	e, ok := w.entities[id]

	// [Start Checks]
	// Make sure the entity exists
	if !ok {
		return false
	}
	// Make sure space is empty
	if _, ok := w.posEntityMap[targetPos]; ok {
		return false
	}
	// [End Checks]

	// Send to observation
	w.onCellUpdate(e.Pos, nil)
	// Remove entity from current position
	delete(w.posEntityMap, e.Pos)

	// Move the entity to new position
	e.Pos = targetPos
	e.Energy -= entityMoveEnergyCost
	w.posEntityMap[targetPos] = e
	// Send to observation
	w.onCellUpdate(e.Pos, e)

	return true
}

// EntityConsume will consume another cell's coccupant
func (w *World) EntityConsume(id int64, targetPos vec2.Vec2) bool {
	// Get the entity by id
	e, ok := w.entities[id]

	// [Start Checks]
	// Make sure the entity exists
	if !ok {
		return false
	}
	// Make sure space is empty
	targetEntity, ok := w.posEntityMap[targetPos]
	if !ok {
		return false
	}
	// Target entity was found, make sure class is consumable
	if targetEntity.Class != "FOOD" {
		return false
	}
	// [End Checks]

	// Remove food entity
	w.DeleteEntity(targetEntity.ID)
	// Add to current entity's energy
	e.Energy += entityConsumeEnergyGain

	return true
}

// EntityLivingCostUpdate performs all living cost calculations, AND removes
//  the entity if it dies returning whether or not it is still alive.
func (w *World) EntityLivingCostUpdate(a *Entity) (isStilAlive bool) {
	// Lower health immediatly if energy is 0
	if a.Energy == 0 {
		a.Health -= entityNoEnergyHealthCost
	}
	// Kill the agent if they have no health and end call
	if a.Health <= 0 {
		w.DeleteEntity(a.ID)
		return false
	}
	// Take away energy
	a.Energy -= entityLivingEnergyCost
	if a.Energy < 0 {
		a.Energy = 0
	}
	return true
}

// DoesEntityExist checks to see if an entity still exists by ID
func (w *World) DoesEntityExist(id int64) bool {
	_, ok := w.entities[id]
	// Return false if an entitiy by that id doesn't exist
	return ok
}
