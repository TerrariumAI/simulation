package environment

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	uuid "github.com/satori/go.uuid"

	firebase "firebase.google.com/go"
	fb "github.com/terrariumai/simulation/pkg/fb"

	"github.com/go-redis/redis"
	"github.com/golang/protobuf/ptypes/empty"
	api "github.com/terrariumai/simulation/pkg/api/environment"
)

const (
	// apiVersion is version of API is provided by server
	apiVersion            = "v1"
	agentLivingEnergyCost = 2
	minFoodBeforeRespawn  = 200
	regionSize            = 16
	maxPositionPadding    = 3
	maxPosition           = 999
)

// toDoServiceServer is implementation of api.ToDoServiceServer proto interface
type environmentServer struct {
	// Environment the server is running in
	env string
	// --- Firebase ---
	// Firebase app
	firebaseApp *firebase.App
	// Mutex to ensure data safety
	m sync.Mutex
	// Redis client
	redisClient *redis.Client
}

// interlockPosition interlocks an x and y value to use as an
// index in redis
func posToRedisIndex(x int32, y int32) (string, error) {
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

func serializeEntity(index string, x int32, y int32, ownerUID string, id string) string {
	return fmt.Sprintf("%s:%v:%v:%s:%s", index, x, y, ownerUID, id)
}

func parseEntityContent(content string) api.Entity {
	values := strings.Split(content, ":")
	println(values)
	x, _ := strconv.Atoi(values[1])
	y, _ := strconv.Atoi(values[2])
	return api.Entity{
		X:        int32(x),
		Y:        int32(y),
		OwnerUID: values[3],
		Id:       values[4],
	}
}

// NewEnvironmentServer creates simulation service
func NewEnvironmentServer(env string) api.EnvironmentServer {
	// initialize redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	pong, err := redisClient.Ping().Result()
	if env != "prod" && err != nil {
		fmt.Println("Couldn't connect to redis: "+pong, err)
		os.Exit(1)
	}
	// initialize server
	s := &environmentServer{
		env:         env,
		firebaseApp: fb.InitializeFirebaseApp(env),
		redisClient: redisClient,
	}

	// // Remove all remote models that were registered for this server before starting
	// removeAllRemoteModelsFromFirebase(s.firebaseApp, s.env)

	return s
}

// Get data for an entity
func (s *environmentServer) CreateEntity(ctx context.Context, req *api.CreateEntityRequest) (*api.CreateEntityResponse, error) {
	// Lock the data, defer unlock until end of call
	s.m.Lock()
	defer s.m.Unlock()

	// Authenticate the user
	user, err := fb.AuthenticateFirebaseAccountWithSecret(ctx, s.firebaseApp, s.env)
	if err != nil {
		return nil, err
	}
	// Make sure the user has supplied data
	if req.Entity == nil {
		return nil, errors.New("Agent not in request")
	}
	// Get an index from the position
	index, err := posToRedisIndex(req.Entity.X, req.Entity.Y)
	if err != nil {
		return nil, err
	}
	// Now we can assume positions are correct sizes
	// (would have thrown an error above if not)
	keys, _, err := s.redisClient.ZScan("entities", 0, index+":*", 0).Result()
	if len(keys) > 0 {
		return nil, errors.New("An entity is already in that position")
	}

	// Create an id for the entity
	entityID := uuid.NewV4().String()
	// Use given ID for testing
	if s.env == "testing" {
		entityID = req.Entity.Id
	}
	// Serialized entity content
	content := serializeEntity(index, req.Entity.X, req.Entity.Y, user["id"].(string), entityID)

	// Add the entity
	err = s.redisClient.ZAdd("entities", redis.Z{
		Score:  float64(0),
		Member: content,
	}).Err()
	if err != nil {
		return nil, err
	}
	// Add the content for later easy indexing
	err = s.redisClient.HSet("entities.content", entityID, content).Err()
	if err != nil {
		return nil, err
	}

	// Return the data for the agent
	return &api.CreateEntityResponse{
		Id: entityID,
	}, nil
}

// Get data for an entity
func (s *environmentServer) GetEntity(ctx context.Context, req *api.GetEntityRequest) (*api.GetEntityResponse, error) {
	// Lock the data, defer unlock until end of call
	s.m.Lock()
	defer s.m.Unlock()
	// Get the content
	hGetEntityContent := s.redisClient.HGet("entities.content", req.Id)
	if hGetEntityContent.Err() != nil {
		return nil, errors.New("Couldn't find an entity by that id")
	}
	content := hGetEntityContent.Val()
	entity := parseEntityContent(content)

	// Return the data for the agent
	return &api.GetEntityResponse{
		Entity: &entity,
	}, nil
}

// Get data for an entity
func (s *environmentServer) DeleteEntity(ctx context.Context, req *api.DeleteEntityRequest) (*api.DeleteEntityResponse, error) {
	// Lock the data, defer unlock until end of call
	s.m.Lock()
	defer s.m.Unlock()

	// Get the content
	hGetEntityContent := s.redisClient.HGet("entities.content", req.Id)
	if hGetEntityContent.Err() != nil {
		return nil, errors.New("Couldn't find an entity by that id")
	}
	content := hGetEntityContent.Val()
	// Parse the content
	entity := parseEntityContent(content)
	// Remove from hash
	delete := s.redisClient.HDel("entities.content", entity.Id)
	if err := delete.Err(); err != nil {
		return nil, fmt.Errorf("Error removing entity: %s", err)
	}
	// Remove from SS
	remove := s.redisClient.ZRem("entities", content)
	if err := remove.Err(); err != nil {
		return nil, fmt.Errorf("Error removing entity: %s", err)
	}
	// Return the data for the agent
	return &api.DeleteEntityResponse{
		Deleted: delete.Val(),
	}, nil
}

// Get data for an entity
func (s *environmentServer) ExecuteAgentAction(ctx context.Context, req *api.ExecuteAgentActionRequest) (*api.ExecuteAgentActionResponse, error) {
	// Lock the data, defer unlock until end of call
	s.m.Lock()
	defer s.m.Unlock()
	// Get the content
	hGetEntityContent := s.redisClient.HGet("entities.content", req.Id)
	if hGetEntityContent.Err() != nil {
		return nil, errors.New("Couldn't find an entity by that id")
	}
	origionalContent := hGetEntityContent.Val()
	// Parse
	entity := parseEntityContent(origionalContent)

	var targetX, targetY = entity.X, entity.Y
	switch req.Direction {
	case 0: // UP
		targetY++
	case 1: // DOWN
		targetY--
	case 2: // LEFT
		targetX--
	case 3: // RIGHT
		targetX++
	}
	// Convert position to index
	index, err := posToRedisIndex(targetX, targetY)
	if err != nil {
		// Return unsuccessful
		return &api.ExecuteAgentActionResponse{
			WasSuccessful: false,
		}, nil
	}
	switch req.Action {
	case 0: // MOVE
		// Check if cell is occupied
		keys, _, _ := s.redisClient.ZScan("entities", 0, index+":*", 0).Result()
		if len(keys) > 0 {
			return nil, errors.New("An entity is already in that position")
		}
		// Cell is clear, move the entity
		content := serializeEntity(index, targetX, targetY, entity.OwnerUID, entity.Id)
		err = s.redisClient.HSet("entities.content", entity.Id, content).Err()
		if err != nil {
			return nil, err
		}
		err = s.redisClient.ZRem("entities", origionalContent).Err()
		if err != nil {
			return nil, err
		}
		err = s.redisClient.ZAdd("entities", redis.Z{
			Score:  float64(0),
			Member: content,
		}).Err()
		if err != nil {
			return nil, err
		}
	default: // INVALID
		return &api.ExecuteAgentActionResponse{
			WasSuccessful: false,
		}, nil
	}

	// Return the data for the agent
	return &api.ExecuteAgentActionResponse{
		WasSuccessful: true,
	}, nil
}

func (s *environmentServer) ResetWorld(ctx context.Context, req *empty.Empty) (*empty.Empty, error) {
	// Lock the data, defer unlock until end of call
	s.m.Lock()
	defer s.m.Unlock()

	// Return
	return &empty.Empty{}, nil
}
