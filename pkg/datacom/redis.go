package datacom

import (
	"errors"
	"fmt"
	"log"

	"github.com/go-redis/redis"
	envApi "github.com/terrariumai/simulation/pkg/api/environment"
)

// IsCellOccupied checks the env to see if a cell has an entity by converting
// the cell position to an index then querying redis
func (dc *Datacom) IsCellOccupied(x int32, y int32) (bool, error) {
	index, err := PosToRedisIndex(x, y)
	if err != nil {
		return true, err
	}

	// Now we can assume positions are correct sizes
	// (would have thrown an error above if not)
	keys, _, err := dc.redisClient.ZScan("entities", 0, index+":*", 0).Result()
	if len(keys) > 0 {
		return true, nil
	}

	return false, nil
}

// CreateEntity sets entity data in the environment. It assumes that
// the location is open and that the owner and model have already been checked.
func (dc *Datacom) CreateEntity(x int32, y int32, class int32, ownerUID string, modelID string, energy int32, health int32, entityID string) error {
	index, err := PosToRedisIndex(x, y)
	if err != nil {
		return err
	}

	// Serialized entity content
	content := SerializeEntity(index, x, y, class, ownerUID, modelID, energy, health, entityID)

	// Add the entity to entities sorted set
	err = dc.redisClient.ZAdd("entities", redis.Z{
		Score:  float64(0),
		Member: content,
	}).Err()
	if err != nil {
		return err
	}
	// Add the content for later easy indexing
	err = dc.redisClient.HSet("entities.content", entityID, content).Err()
	if err != nil {
		return err
	}
	// Add the entitiy to the model's data
	err = dc.redisClient.SAdd("model:"+modelID+":entities", entityID).Err()
	if err != nil {
		return err
	}

	// Send update
	dc.PublishEvent("createEntity", envApi.Entity{
		Id:       entityID,
		X:        x,
		Y:        y,
		Class:    class,
		Energy:   energy,
		Health:   health,
		OwnerUID: ownerUID,
		ModelID:  modelID,
	})

	return nil
}

// UpdateEntity updates an entity. It first removes the origional entity
// data then creates new entity data and index from the given params and
// writes those.
func (dc *Datacom) UpdateEntity(origionalContent string, x int32, y int32, class int32, ownerUID string, modelID string, energy int32, health int32, entityID string) error {
	index, err := PosToRedisIndex(x, y)
	if err != nil {
		return err
	}
	content := SerializeEntity(index, x, y, class, ownerUID, modelID, energy, health, entityID)
	err = dc.redisClient.HSet("entities.content", entityID, content).Err()
	if err != nil {
		return err
	}
	err = dc.redisClient.ZRem("entities", origionalContent).Err()
	if err != nil {
		return err
	}
	err = dc.redisClient.ZAdd("entities", redis.Z{
		Score:  float64(0),
		Member: content,
	}).Err()

	// Send update
	dc.PublishEvent("updateEntity", envApi.Entity{
		Id:       entityID,
		X:        x,
		Y:        y,
		Class:    class,
		Energy:   energy,
		Health:   health,
		OwnerUID: ownerUID,
		ModelID:  modelID,
	})

	return nil
}

// GetEntity gets an entity from the environment by id
func (dc *Datacom) GetEntity(id string) (*envApi.Entity, *string, error) {
	// Get the content
	hGetEntityContent := dc.redisClient.HGet("entities.content", id)
	if hGetEntityContent.Err() != nil {
		return nil, nil, errors.New("Couldn't find an entity by that id")
	}
	content := hGetEntityContent.Val()
	entity, _ := ParseEntityContent(content)

	// Send update
	dc.PublishEvent("createEntity", entity)

	return &entity, &content, nil
}

// DeleteEntity completely removes an entity from existence from the environment
func (dc *Datacom) DeleteEntity(id string) (int64, error) {
	// Get the content
	hGetEntityContent := dc.redisClient.HGet("entities.content", id)
	if hGetEntityContent.Err() != nil {
		return 0, errors.New("Error deleting entity: Couldn't find an entity by that id")
	}
	content := hGetEntityContent.Val()
	// Parse the content
	entity, _ := ParseEntityContent(content)
	// Remove from hash
	delete := dc.redisClient.HDel("entities.content", entity.Id)
	if err := delete.Err(); err != nil {
		return 0, fmt.Errorf("Error deleting entity: %s", err)
	}
	// Remove from SS
	remove := dc.redisClient.ZRem("entities", content)
	if err := remove.Err(); err != nil {
		return 0, fmt.Errorf("Error deleting entity: %s", err)
	}
	// Remove from model
	err := dc.redisClient.SRem("model:"+entity.ModelID+":entities", entity.Id).Err()
	if err != nil {
		return 0, err
	}

	// Send update
	dc.PublishEvent("deleteEntity", envApi.Entity{
		Id: id,
	})

	return delete.Val(), nil
}

// GetEntitiesForModel gets a list of entities for a specific model
func (dc *Datacom) GetEntitiesForModel(modelID string) ([]interface{}, error) {
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
	return entitiesContent, nil
}

// GetEntitiesAroundPosition gets the entities directly around a position
func (dc *Datacom) GetEntitiesAroundPosition(xMin int32, yMin int32, xMax int32, yMax int32) ([]string, error) {
	indexMin, err := PosToRedisIndex(xMin, yMin)
	if err != nil {
		return nil, fmt.Errorf("Error converting min/max positions to index: %v", err)
	}
	indexMax, err := PosToRedisIndex(xMax, yMax)
	if err != nil {
		return nil, fmt.Errorf("Error converting min/max positions to index: %v", err)
	}
	// Perform the query
	rangeQuery := dc.redisClient.ZRangeByLex("entities", redis.ZRangeBy{
		Min: "(" + indexMin,
		Max: "(" + indexMax,
	})
	if err := rangeQuery.Err(); err != nil {
		return nil, fmt.Errorf("Error in range query: %v", err)
	}
	closeEntitiesContent := rangeQuery.Val()
	return closeEntitiesContent, nil
}

// GetEntitiesInRegion returns the entities in a specific region
func (dc *Datacom) GetEntitiesInRegion(x int32, y int32) ([]*envApi.Entity, error) {
	entities := []*envApi.Entity{}

	xMin := x * regionSize
	yMin := y * regionSize
	xMax := xMin + regionSize
	yMax := yMin + regionSize
	// Convert positions to index
	indexMin, err := PosToRedisIndex(xMin, yMin)
	if err != nil {
		return nil, fmt.Errorf("Error converting min/max positions to index: %v", err)
	}
	indexMax, err := PosToRedisIndex(xMax, yMax)
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
		entitiy, _ := ParseEntityContent(content)
		entities = append(entities, &entitiy)
	}

	return entities, nil
}
