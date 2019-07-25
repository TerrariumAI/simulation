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
func (dc *Datacom) IsCellOccupied(x uint32, y uint32) (bool, error) {
	index, err := posToRedisIndex(x, y)
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
func (dc *Datacom) CreateEntity(e envApi.Entity) error {
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
	entity, _ := parseEntityContent(content)
	dc.pubsub.PublishEvent("createEntity", entity)

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
	dc.pubsub.PublishEvent("updateEntity", e)

	return nil
}

// GetEntity gets an entity from the environment by id
func (dc *Datacom) GetEntity(id string) (*envApi.Entity, *string, error) {
	// Get the content
	hGetEntityContent := dc.redisClient.HGet("entities.content", id)
	if hGetEntityContent.Err() != nil {
		return nil, nil, errors.New("entity does not exist")
	}
	content := hGetEntityContent.Val()
	entity, _ := parseEntityContent(content)

	return &entity, &content, nil
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
	dc.pubsub.PublishEvent("deleteEntity", entity)

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
	// If the entity is out of bounds for some reason, delete it
	// TODO - remove this, make it so it is impossible to get here in the first place
	if entity.X < 1 || entity.Y < 1 {
		dc.DeleteEntity(entity.Id)
		err := errors.New("entity was invalid and has been deleted")
		log.Printf("ERROR: %v\n", err)
		return nil, err
	}

	obsv := collectiveApi.Observation{
		Id: entity.Id,
	}
	xMin := entity.X - 1
	xMax := entity.X + 1
	yMin := entity.Y - 1
	yMax := entity.Y + 1
	// Query for entities near this position
	closeEntitiesContent, err := dc.getEntitiesInArea(xMin, yMin, xMax, yMax)
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
	for y = int32(entity.Y) + 1; y >= int32(entity.Y)-1; y-- {
		for x = int32(entity.X) - 1; x <= int32(entity.X)+1; x++ {
			otherIndex, err := posToRedisIndex(uint32(x), uint32(y))
			if err != nil {
				return nil, err
			}
			if otherIndex == index {
				continue
			}
			if x < minPosition || y < minPosition {
				// If the position is out of bounds, return a rock
				obsv.Cells = append(obsv.Cells, &collectiveApi.Entity{Id: "", Class: 2})
				continue
			}
			if otherEntity, ok := indexEntityMap[otherIndex]; ok {
				obsv.Cells = append(obsv.Cells, &collectiveApi.Entity{Id: otherEntity.Id, Class: otherEntity.Class})
			} else {
				obsv.Cells = append(obsv.Cells, &collectiveApi.Entity{Id: "", Class: 0})
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
