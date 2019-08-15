package datacom

import (
	"errors"
	"fmt"
	"log"

	"github.com/go-redis/redis"
	collectiveApi "github.com/terrariumai/simulation/pkg/api/collective"
	envApi "github.com/terrariumai/simulation/pkg/api/environment"
)

// IsCellOccupied checks the env to see if a cell has an entity by converting
// the cell position to an index then querying redis
func (dc *Datacom) IsCellOccupied(x uint32, y uint32) (bool, *envApi.Entity, string, error) {
	index, err := posToRedisIndex(x, y)
	if err != nil {
		// TODO - Returning an empty string here may cause errors down the line
		return true, nil, "", err
	}

	// Now we can assume positions are correct sizes
	// (would have thrown an error above if not)
	keys, _, err := dc.redisClient.ZScan("entities", 0, index+":*", 0).Result()
	if len(keys) > 0 {
		e, _ := parseEntityContent(keys[0])
		return true, &e, keys[0], nil
	}

	return false, nil, "", nil
}

// CreateEntity sets entity data in the environment. It assumes that
// the location is open and that the owner and model have already been checked.
func (dc *Datacom) CreateEntity(e envApi.Entity, shouldPublish bool) error {
	// Serialized entity content
	content, err := serializeEntity(e)
	if err != nil {
		log.Println("ERROR: ", err)
		return err
	}

	// Add the entity to entities sorted set
	err = dc.redisClient.ZAdd("entities", redis.Z{
		Score:  float64(0),
		Member: content,
	}).Err()
	if err != nil {
		return err
	}
	// Add the content for later easy indexing
	err = dc.redisClient.HSet("entities.content", e.Id, content).Err()
	if err != nil {
		return err
	}
	// Add the entitiy to the model's data
	err = dc.redisClient.SAdd("model:"+e.ModelID+":entities", e.Id).Err()
	if err != nil {
		return err
	}

	// Send update
	if shouldPublish {
		dc.pubsub.QueuePublishEvent("createEntity", &e, e.X, e.Y)
	}

	return nil
}

// UpdateEntity updates an entity. It first removes the origional entity
// data then creates new entity data and index from the given params and
// writes those.
func (dc *Datacom) UpdateEntity(origionalContent string, e envApi.Entity) error {
	content, err := serializeEntity(e)
	if err != nil {
		log.Println("ERROR: ", err)
		return err
	}
	err = dc.redisClient.HSet("entities.content", e.Id, content).Err()
	if err != nil {
		log.Println("ERROR: ", err)
		return err
	}
	err = dc.redisClient.ZRem("entities", origionalContent).Err()
	if err != nil {
		log.Println("ERROR: ", err)
		return err
	}
	err = dc.redisClient.ZAdd("entities", redis.Z{
		Score:  float64(0),
		Member: content,
	}).Err()

	// Send update
	dc.pubsub.QueuePublishEvent("updateEntity", &e, e.X, e.Y)

	return nil
}

// GetEntity gets an entity from the environment by id
func (dc *Datacom) GetEntity(id string) (*envApi.Entity, string, error) {
	// Get the content
	hGetEntityContent := dc.redisClient.HGet("entities.content", id)
	if hGetEntityContent.Err() != nil {
		return nil, "", errors.New("entity does not exist")
	}
	content := hGetEntityContent.Val()
	entity, _ := parseEntityContent(content)

	return &entity, content, nil
}

// DeleteEntity completely removes an entity from existence from the environment
func (dc *Datacom) DeleteEntity(id string) (int64, error) {
	// Get the content
	hGetEntityContent := dc.redisClient.HGet("entities.content", id)
	if err := hGetEntityContent.Err(); err != nil {
		log.Printf("ERROR: %v\n", err)
		return 0, err
	}
	content := hGetEntityContent.Val()
	// Parse the content
	entity, _ := parseEntityContent(content)
	// Remove from hash
	delete := dc.redisClient.HDel("entities.content", entity.Id)
	if err := delete.Err(); err != nil {
		log.Printf("ERROR: %v\n", err)
		return 0, err
	}
	// Remove from SS
	remove := dc.redisClient.ZRem("entities", content)
	if err := remove.Err(); err != nil {
		log.Printf("ERROR: %v\n", err)
		return 0, err
	}
	// Remove from model
	err := dc.redisClient.SRem("model:"+entity.ModelID+":entities", entity.Id).Err()
	if err != nil {
		log.Printf("ERROR: %v\n", err)
		return 0, err
	}

	// Send update
	dc.pubsub.QueuePublishEvent("deleteEntity", &entity, entity.X, entity.Y)

	return delete.Val(), nil
}

// GetEntitiesForModel gets a list of entities for a specific model
func (dc *Datacom) GetEntitiesForModel(modelID string) ([]envApi.Entity, error) {
	// Get the entitiy IDs for this model
	entityIdsRequest := dc.redisClient.SMembers("model:" + modelID + ":entities")
	if err := entityIdsRequest.Err(); err != nil {
		log.Fatalf("ConnectRemoteModel(): %v", err)
		return nil, errors.New("ConnectRemoteModel(): Couldn't access the database to get the entity ids for this model")
	}
	entityIds := entityIdsRequest.Val()
	// Get the entities for this model
	entitiesContentRequest := dc.redisClient.HMGet("entities.content", entityIds...)
	// Get the content for each entity
	entitiesContent := make([]interface{}, 0)
	if len(entityIds) > 0 {
		if err := entitiesContentRequest.Err(); err != nil {
			return nil, fmt.Errorf("ConnectRemoteModel(): Couldn't access the database to get the entities: %v", err)
		}
		entitiesContent = entitiesContentRequest.Val()
	}
	// Convert content to entities
	entities := []envApi.Entity{}
	for _, content := range entitiesContent {
		entity, _ := parseEntityContent(content.(string))
		entities = append(entities, entity)
	}
	return entities, nil
}

// getEntitiesAroundPosition gets the entities directly around a position
// TODO - this doesn't work correctly as there isn't enough fine tunement to do this. Use another
// method
func (dc *Datacom) getEntitiesInArea(xMin uint32, yMin uint32, xMax uint32, yMax uint32) ([]string, error) {
	indexMin, err := posToRedisIndex(xMin, yMin)
	if err != nil {
		return nil, fmt.Errorf("Error converting min/max positions to index: %v", err)
	}
	indexMax, err := posToRedisIndex(xMax, yMax)
	if err != nil {
		return nil, fmt.Errorf("Error converting min/max positions to index: %v", err)
	}
	// Perform the query
	rangeQuery := dc.redisClient.ZRangeByLex("entities", redis.ZRangeBy{
		Min: "[" + indexMin,
		Max: "[" + indexMax,
	})
	if err := rangeQuery.Err(); err != nil {
		return nil, fmt.Errorf("Error in range query: %v", err)
	}
	entitiesContent := rangeQuery.Val()
	return entitiesContent, nil
}

// GetObservationForEntity returns observations for a specific entity
func (dc *Datacom) GetObservationForEntity(entity envApi.Entity) (*collectiveApi.Observation, error) {
	content, err := serializeEntity(entity)
	if err != nil {
		log.Printf("ERROR: %v\n", err)
		return nil, err
	}
	index, err := posToRedisIndex(entity.X, entity.Y)
	if err != nil {
		log.Printf("ERROR: %v\n", err)
		return nil, err
	}

	obsv := collectiveApi.Observation{
		Id:      entity.Id,
		Energy:  entity.Energy,
		Health:  entity.Health,
		IsAlive: true,
	}
	xMin := int32(entity.X) - dc.EntityVisionDist
	xMax := int32(entity.X) + dc.EntityVisionDist
	yMin := int32(entity.Y) - dc.EntityVisionDist
	yMax := int32(entity.Y) + dc.EntityVisionDist
	// Make sure we are only querying valid positions
	if xMin < minPosition {
		xMin = minPosition
	}
	if yMin < minPosition {
		yMin = minPosition
	}
	if xMax > maxPosition {
		xMax = maxPosition
	}
	if yMax > maxPosition {
		yMax = maxPosition
	}
	// Query for entities near this position
	closeEntitiesContent, err := dc.getEntitiesInArea(uint32(xMin), uint32(yMin), uint32(xMax), uint32(yMax))
	if err != nil {
		log.Printf("ERROR: %v\n", err)
		return nil, err
	}
	// Add all the other entities to the indexEntityMap
	// Match them up with the correct positions
	indexEntityMap := make(map[string]envApi.Entity)
	for _, otherContent := range closeEntitiesContent {
		// Don't count the same entity
		if content == otherContent {
			continue
		}
		otherEntity, index := parseEntityContent(otherContent)
		indexEntityMap[index] = otherEntity
	}
	var x int32
	var y int32
	for y = int32(entity.Y) + dc.EntityVisionDist; y >= int32(entity.Y)-dc.EntityVisionDist; y-- {
		for x = int32(entity.X) - dc.EntityVisionDist; x <= int32(entity.X)+dc.EntityVisionDist; x++ {
			// If position is invalid, set it to untraversable entity (rock)
			if x < minPosition || x > maxPosition || y < minPosition || y > maxPosition {
				obsv.Sight = append(obsv.Sight, &collectiveApi.Entity{Id: "", ClassID: 2})
				continue
			}
			// Attempt to get redis index from position
			otherIndex, err := posToRedisIndex(uint32(x), uint32(y))
			if err != nil { // If it errors here, something went wrong with logic above
				return nil, err
			}
			// If the index is the same as our current entity
			if otherIndex == index {
				continue
			}
			if otherEntity, ok := indexEntityMap[otherIndex]; ok {
				obsv.Sight = append(obsv.Sight, &collectiveApi.Entity{Id: otherEntity.Id, ClassID: otherEntity.ClassID})
			} else {
				obsv.Sight = append(obsv.Sight, &collectiveApi.Entity{Id: "", ClassID: 0})
			}
		}
	}
	return &obsv, nil
}

// GetEntitiesInRegion returns the entities in a specific region
func (dc *Datacom) GetEntitiesInRegion(x uint32, y uint32) ([]*envApi.Entity, error) {
	entities := []*envApi.Entity{}

	xMin := x * regionSize
	yMin := y * regionSize
	xMax := xMin + regionSize
	yMax := yMin + regionSize
	// Convert positions to index
	indexMin, err := posToRedisIndex(xMin, yMin)
	if err != nil {
		return nil, fmt.Errorf("Error converting min/max positions to index: %v", err)
	}
	indexMax, err := posToRedisIndex(xMax, yMax)
	if err != nil {
		return nil, fmt.Errorf("Error converting min/max positions to index: %v", err)
	}
	// Perform the query
	rangeQuery := dc.redisClient.ZRangeByLex("entities", redis.ZRangeBy{
		Min: "[" + indexMin,
		Max: "(" + indexMax,
	})
	if err := rangeQuery.Err(); err != nil {
		return nil, fmt.Errorf("Error in range query: %v", err)
	}
	entitiesContent := rangeQuery.Val()

	for _, content := range entitiesContent {
		entitiy, _ := parseEntityContent(content)
		entities = append(entities, &entitiy)
	}

	return entities, nil
}

// --------------
// Pheromones
// --------------

// SetPheromone sets a pheromone at a specific index (position). We don't
// create/update/delete because pheromones can be updated
func (dc *Datacom) SetPheromone(origionalContent string, p envApi.Pheromone) error {
	content, err := serializePheromone(p)
	index, _ := posToRedisIndex(p.X, p.Y)
	if err != nil {
		log.Println("ERROR: ", err)
		return err
	}
	err = dc.redisClient.ZRem("pheromones", index+":*").Err()
	if err != nil {
		log.Println("ERROR: ", err)
		return err
	}
	err = dc.redisClient.ZAdd("pheromones", redis.Z{
		Score:  float64(0),
		Member: content,
	}).Err()

	// Send update
	dc.pubsub.QueuePublishEvent("setPheromone", &p, p.X, p.Y)

	return nil
}
