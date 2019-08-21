package datacom

import (
	"errors"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis"
	collectiveApi "github.com/terrariumai/simulation/pkg/api/collective"
	envApi "github.com/terrariumai/simulation/pkg/api/environment"
)

func zipStrings(str1 string, str2 string) string {
	res := ""
	for i := 0; i < len(str1); i++ {
		res = res + str1[i:i+1] + str2[i:i+1]
	}
	return res
}

func rjust(s string, filler byte, max int) string {
	if len(s) < max {
		return rjust(string(filler)+s, filler, max)
	}
	return s
}

// constructSpacequeryCalls constructs zrangeby calls for zrangebylex queries
func constructSpacequeryCalls(x0 uint32, y0 uint32, x1 uint32, y1 uint32, exp float64) []redis.ZRangeBy {
	calls := []redis.ZRangeBy{}

	bits := int(exp * 2)
	xStart := x0 / uint32(math.Pow(2, exp))
	xEnd := x1 / uint32(math.Pow(2, exp))
	yStart := y0 / uint32(math.Pow(2, exp))
	yEnd := y1 / uint32(math.Pow(2, exp))
	for x := xStart; x <= xEnd; x++ {
		for y := yStart; y <= yEnd; y++ {
			xRangeStart := x * uint32(math.Pow(2, exp))
			yRangeStart := y * uint32(math.Pow(2, exp))

			xBin := strconv.FormatUint(uint64(xRangeStart), 2)
			xBin = rjust(xBin, '0', 9)
			yBin := strconv.FormatUint(uint64(yRangeStart), 2)
			yBin = rjust(yBin, '0', 9)

			s := zipStrings(xBin, yBin)
			e := s[:len(s)-bits] + strings.Repeat("1", bits)
			calls = append(calls, redis.ZRangeBy{
				Min: "[" + s,
				Max: "[" + e,
			})
		}
	}

	return calls
}

// spacequery will make ain INACCURATE query within a space. It is possible,
// and very likely that there will be elements included that are outside the
// space, but will always include all elements within the space.
func (dc *Datacom) spacequery(key string, x0 uint32, y0 uint32, x1 uint32, y1 uint32) ([]string, error) {
	calls := constructSpacequeryCalls(x0, y0, x1, y1, maxPositionCharLength)

	// Perform queries to get the content array
	combinedContentArray := []string{}
	for _, call := range calls {
		// Perform the query
		rangeQuery := dc.redisClient.ZRangeByLex(key, call)
		if err := rangeQuery.Err(); err != nil {
			return nil, fmt.Errorf("Error in range query: %v", err)
		}
		contentArray := rangeQuery.Val()
		combinedContentArray = append(combinedContentArray, contentArray...)
	}

	return combinedContentArray, nil
}

// --------------
// Entities
// --------------

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

// GetObservationForEntity returns observations for a specific entity
func (dc *Datacom) GetObservationForEntity(entity envApi.Entity) (*collectiveApi.Observation, error) {
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

	// Query for entities near this position
	// Note: we handle grabbing specific entities below, can ignore extras
	x0, y0, x1, y1 := calcSpaceAroundPoint(int32(entity.X), int32(entity.Y), dc.EntityVisionDist)
	closeEntities, err := dc.GetEntitiesInSpace(uint32(x0), uint32(y0), uint32(x1), uint32(y1))
	if err != nil {
		log.Printf("ERROR: %v\n", err)
		return nil, err
	}
	// Query for effects near this position
	// Note: we handle grabbing specific entities below, can ignore extras
	x0, y0, x1, y1 = calcSpaceAroundPoint(int32(entity.X), int32(entity.Y), dc.EntityVisionDist)
	closeEffects, err := dc.GetEffectsInSpace(uint32(x0), uint32(y0), uint32(x1), uint32(y1))
	if err != nil {
		log.Printf("ERROR: %v\n", err)
		return nil, err
	}

	// Add all the other entities to the indexEntityMap
	// Match them up with the correct positions
	indexEntityMap := make(map[string]envApi.Entity)
	for _, otherEntity := range closeEntities {
		index, _ := posToRedisIndex(otherEntity.X, otherEntity.Y)
		indexEntityMap[index] = *otherEntity
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

	// Add all the other entities to the indexEntityMap
	// Match them up with the correct positions
	indexEffectMap := make(map[string]envApi.Effect)
	for _, effect := range closeEffects {
		index, _ := posToRedisIndex(effect.X, effect.Y)
		indexEffectMap[index] = *effect
	}
	for y = int32(entity.Y) + dc.EntitySmellDist; y >= int32(entity.Y)-dc.EntitySmellDist; y-- {
		for x = int32(entity.X) - dc.EntitySmellDist; x <= int32(entity.X)+dc.EntitySmellDist; x++ {
			println("asdfasdf", x, y)
			// If position is invalid, set it to untraversable entity (rock)
			if x < minPosition || x > maxPosition || y < minPosition || y > maxPosition {
				obsv.Smell = append(obsv.Smell, &collectiveApi.Effect{ClassID: collectiveApi.Effect_Class(0)})
				continue
			}
			// Attempt to get redis index from position
			otherIndex, err := posToRedisIndex(uint32(x), uint32(y))
			if err != nil { // If it errors here, something went wrong with logic above
				return nil, err
			}
			if effect, ok := indexEffectMap[otherIndex]; ok {
				strength := uint32(100 / math.Pow(float64(effect.Decay), float64(time.Now().Unix()-effect.Timestamp)))
				obsv.Smell = append(obsv.Smell, &collectiveApi.Effect{ClassID: collectiveApi.Effect_Class(effect.ClassID), Value: effect.Value, Strength: strength})
			} else {
				obsv.Smell = append(obsv.Smell, &collectiveApi.Effect{ClassID: collectiveApi.Effect_Class(0)})
			}
		}
	}
	return &obsv, nil
}

// GetEntitiesInSpace returns the entities in a specific region
func (dc *Datacom) GetEntitiesInSpace(x0 uint32, y0 uint32, x1 uint32, y1 uint32) ([]*envApi.Entity, error) {
	entities := []*envApi.Entity{}

	// Perform the query
	contentArray, err := dc.spacequery("entities", x0, y0, x1, y1)
	if err != nil {
		return nil, fmt.Errorf("Error converting min/max positions to index: %v", err)
	}

	for _, content := range contentArray {
		entity, _ := parseEntityContent(content)
		// Ignore entities outside space
		if entity.X < x0 || entity.X > x1 || entity.X < y0 || entity.X > y1 {
			continue
		}
		entities = append(entities, &entity)
	}

	return entities, nil
}

// --------------
// Effects
// --------------

// CreateEffect sets an effect at a specific index (position)
func (dc *Datacom) CreateEffect(effect envApi.Effect) error {
	if effect.Timestamp == 0 {
		effect.Timestamp = time.Now().Unix()
	}
	content, err := serializeEffect(effect)
	fmt.Println(content)
	if err != nil {
		log.Println("ERROR: ", err)
		return err
	}
	err = dc.redisClient.ZAdd("effects", redis.Z{
		Score:  float64(0),
		Member: content,
	}).Err()

	// Send update
	dc.pubsub.QueuePublishEvent("createEffect", &effect, effect.X, effect.Y)

	return nil
}

// GetEffectsInSpace returns the effects in a specific region
func (dc *Datacom) GetEffectsInSpace(x0 uint32, y0 uint32, x1 uint32, y1 uint32) ([]*envApi.Effect, error) {
	effects := []*envApi.Effect{}

	// Perform the query
	contentArray, err := dc.spacequery("effects", x0, y0, x1, y1)
	if err != nil {
		return nil, fmt.Errorf("Error converting min/max positions to index: %v", err)
	}

	for _, content := range contentArray {
		effect, _ := parseEffectContent(content)
		// Ignore outside
		if effect.X < x0 || effect.X > x1 || effect.Y < y0 || effect.Y > y1 {
			continue
		}
		// Calculate strength
		strength := uint32(100 / math.Pow(float64(effect.Decay), float64(time.Now().Unix()-effect.Timestamp)))
		if strength <= effect.DelThresh {
			dc.DeleteEffect(effect)
			continue
		}
		effects = append(effects, &effect)
	}

	return effects, nil
}

// DeleteEffect removes an effect
func (dc *Datacom) DeleteEffect(effect envApi.Effect) (int64, error) {
	content, _ := serializeEffect(effect)
	// Remove from SS
	remove := dc.redisClient.ZRem("effects", content)
	if err := remove.Err(); err != nil {
		log.Printf("ERROR: %v\n", err)
		return 0, err
	}

	// Send update
	dc.pubsub.QueuePublishEvent("deleteEffect", &effect, effect.X, effect.Y)

	return remove.Val(), nil
}
