package v1

// Entity - data for entities that exist in cells
type Entity struct {
	id     int64
	class  string
	pos    Vec2
	energy int32
	health int32
}

const initialEnergy = 100
const initialHealth = 100

// Create a new entity and add it to the simulation
func (s *simulationServiceServer) NewEntity(class string, pos Vec2) *Entity {
	// Create the entity
	id := s.nextEntityID
	s.nextEntityID++
	e := Entity{id, class, pos, initialEnergy, initialHealth}
	s.entities[id] = &e
	s.posEntityMap[pos] = &e

	// Broadcast update
	s.BroadcastCellUpdate(e.pos, &e, "")

	return &e
}

// Remove an entity by Id and broadcast the update
func (s *simulationServiceServer) RemoveEntityByID(id int64) bool {
	// Get the entitiy
	e, ok := s.entities[id]
	// Return false if an entitiy by that id doesn't exist
	if !ok {
		return false
	}
	// Remove the entity
	delete(s.entities, e.id)
	delete(s.posEntityMap, e.pos)
	// Broadcast update
	s.BroadcastCellUpdate(e.pos, nil, "")

	return true
}

// Move an entity
func (s *simulationServiceServer) EntityMove(id int64, targetPos Vec2) bool {
	// Get the entity by id
	e, ok := s.entities[id]

	// [Start Checks]
	// Make sure the entity exists
	if !ok {
		return false
	}
	// Make sure space is empty
	if _, ok := s.posEntityMap[targetPos]; ok {
		return false
	}
	// [End Checks]

	// Send to observation
	s.BroadcastCellUpdate(e.pos, nil, "")
	// Remove entity from current position
	delete(s.posEntityMap, e.pos)

	// Move the entity to new position
	e.pos = targetPos
	s.posEntityMap[targetPos] = e
	// Send to observation
	s.BroadcastCellUpdate(e.pos, e, "")

	return true
}

// Entity consume another cell's coccupant
func (s *simulationServiceServer) EntityConsume(id int64, targetPos Vec2) bool {
	// Get the entity by id
	e, ok := s.entities[id]

	// [Start Checks]
	// Make sure the entity exists
	if !ok {
		return false
	}
	// Make sure space is empty
	targetEntity, ok := s.posEntityMap[targetPos]
	if !ok {
		return false
	} else {
		if targetEntity.class != "FOOD" {
			return false
		}
	}
	// [End Checks]

	// Remove food entity
	s.RemoveEntityByID(targetEntity.id)
	// Add to current entity's energy
	e.energy += 10

	return true
}

func (s *simulationServiceServer) isCellOccupied(pos Vec2) bool {
	if _, ok := s.posEntityMap[pos]; ok {
		return true
	}
	return false
}
