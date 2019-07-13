package datacom

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"

	firebase "firebase.google.com/go"
	"github.com/go-redis/redis"
	envApi "github.com/terrariumai/simulation/pkg/api/environment"
	"google.golang.org/api/option"
)

const (
	serviceAccountProdFileLocation    = "./serviceAccountKey.json"
	serviceAccountStagingFileLocation = "./serviceAccountKey_staging.json"

	mockSecret = "MOCK-SECRET"

	maxPositionPadding = 3
	maxPosition        = 999
)

// Datacom is an object that makes it easy to communicate with our
// databases. It handles figuring out where each specific
// bit of data is (redis, firebase, etc.) and how to access it (auth).
type Datacom struct {
	// current envirinment
	env string
	// firebase client
	firebaseApp *firebase.App
	// redis client
	redisClient *redis.Client
}

// RemoteModel struct for parsing and storing RM data from databases
type RemoteModel struct {
	id      string
	ownerID string
	name    string
}

// PosToRedisIndex interlocks an x and y value to use as an
// index in redis
func PosToRedisIndex(x int32, y int32) (string, error) {
	// negatives are not allowed
	if x < 0 || y < 0 || x > maxPosition || y > maxPosition {
		return "", errors.New("Invalid position")
	}
	xString := strconv.Itoa(int(x))
	yString := strconv.Itoa(int(y))
	interlocked := ""
	// make sure x and y are the correct length when converted to str
	if len(xString) > maxPositionPadding || len(yString) > maxPositionPadding {
		return "", errors.New("X or Y position are too large")
	}
	// add padding
	for len(xString) < maxPositionPadding {
		xString = "0" + xString
	}
	for len(yString) < maxPositionPadding {
		yString = "0" + yString
	}
	// interlock
	for i := 0; i < maxPositionPadding; i++ {
		interlocked = interlocked + xString[i:i+1] + yString[i:i+1]
	}

	return interlocked, nil
}

// SerializeEntity takes in all the values for an entity and serializes them
//  to an entity content
func SerializeEntity(index string, x int32, y int32, class int32, ownerUID string, modelID string, id string) string {
	return fmt.Sprintf("%s:%v:%v:%v:%s:%s:%s", index, x, y, class, ownerUID, modelID, id)
}

// ParseEntityContent takes entity content and parses it out to an entity
func ParseEntityContent(content string) (envApi.Entity, string) {
	values := strings.Split(content, ":")
	x, _ := strconv.Atoi(values[1])
	y, _ := strconv.Atoi(values[2])
	class, _ := strconv.Atoi(values[3])
	return envApi.Entity{
		X:        int32(x),
		Y:        int32(y),
		Class:    int32(class),
		OwnerUID: values[4],
		ModelID:  values[5],
		Id:       values[6],
	}, values[0]
}

// NewDatacom instantiates a new datacom object with proper clients
// according to the environment
func NewDatacom(env string) (*Datacom, error) {
	dc := &Datacom{
		env: env,
	}

	redisAddr := "localhost:6379"
	if env == "prod" {
		// Load env vars
		err := godotenv.Load()
		if err != nil {
			log.Fatal("Error loading .env file")
		}
		redisAddr = os.Getenv("REDIS_ADDR")
	}

	// If we are training, we don't ever connect to any servers
	if env == "training" {
		return dc, nil
	}

	// Setup Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	_, err := redisClient.Ping().Result()
	if err != nil {
		return nil, err
	}
	dc.redisClient = redisClient

	// Setup Firebase
	switch env {
	case "staging":
		// FIREBASE STAGING
		opt := option.WithCredentialsFile(serviceAccountStagingFileLocation)
		app, err := firebase.NewApp(context.Background(), nil, opt)
		if err != nil {
			return nil, err
		}
		dc.firebaseApp = app
	case "prod":
		// FIREBASE PROD
		opt := option.WithCredentialsFile(serviceAccountProdFileLocation)
		app, err := firebase.NewApp(context.Background(), nil, opt)
		if err != nil {
			return nil, err
		}
		dc.firebaseApp = app
	}

	return dc, nil
}

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
func (dc *Datacom) CreateEntity(x int32, y int32, class int32, ownerUID string, modelID string, entityID string) error {
	index, err := PosToRedisIndex(x, y)
	if err != nil {
		return err
	}

	// Serialized entity content
	content := SerializeEntity(index, x, y, class, ownerUID, modelID, entityID)

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

	return nil
}

// UpdateEntity updates an entity. It first removes the origional entity
// data then creates new entity data and index from the given params and
// writes those.
func (dc *Datacom) UpdateEntity(origionalContent string, x int32, y int32, class int32, ownerUID string, modelID string, entityID string) error {
	index, err := PosToRedisIndex(x, y)
	if err != nil {
		return err
	}
	content := SerializeEntity(index, x, y, class, ownerUID, modelID, entityID)
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

// --------------
// FIREBASE
// --------------

// GetRemoteModelMetadataForUser checks the database to see if a remote model exists,
// if so returns metadata
func (dc *Datacom) GetRemoteModelMetadataForUser(modelID string, userID string) (*RemoteModel, error) {
	// Init client
	ctx := context.Background()
	client, err := dc.firebaseApp.Firestore(ctx)
	defer client.Close()
	if err != nil {
		return nil, err
	}
	// Try to get the RM
	dsnap, err := client.Collection("remoteModels").Doc(modelID).Get(ctx)
	if err != nil {
		return nil, err
	}
	var remoteModel RemoteModel
	dsnap.DataTo(&remoteModel)
	remoteModel.id = modelID

	// Check if this is the correct owner
	if remoteModel.ownerID != userID {
		return nil, errors.New("That RM does not belong to you")
	}

	return &remoteModel, nil
}
