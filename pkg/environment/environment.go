package environment

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"

	uuid "github.com/satori/go.uuid"

	firebase "firebase.google.com/go"

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
	if x < 0 || y < 0 {
		return "", errors.New("Position cannot be negative")
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

func serializeEntity(index string, x int32, y int32, ownerID string, id string) string {
	return fmt.Sprintf("%s:%v:%v:%s:%s", index, x, y, ownerID, id)
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
		firebaseApp: initializeFirebaseApp(env),
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
	user, err := authenticateFirebaseAccountWithSecret(ctx, s.firebaseApp, s.env)
	if err != nil {
		return nil, err
	}
	// Make sure the user has supplied data
	if req.Agent == nil {
		return nil, errors.New("Agent not in request")
	}
	// Get an index from the position
	index, err := posToRedisIndex(req.Agent.X, req.Agent.Y)
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
	// Serialized entity content
	content := serializeEntity(index, req.Agent.X, req.Agent.Y, user["id"].(string), entityID)

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
		Id: 0,
	}, nil
}

// Get data for an entity
func (s *environmentServer) GetEntity(ctx context.Context, req *api.GetEntityRequest) (*api.GetEntityResponse, error) {
	// Lock the data, defer unlock until end of call
	s.m.Lock()
	defer s.m.Unlock()

	// Return the data for the agent
	return &api.GetEntityResponse{
		Entity: &api.Entity{
			Id:    0,
			Class: "AGENT",
		},
	}, nil
}

// Get data for an entity
func (s *environmentServer) DeleteEntity(ctx context.Context, req *api.DeleteEntityRequest) (*api.DeleteEntityResponse, error) {
	// Lock the data, defer unlock until end of call
	s.m.Lock()
	defer s.m.Unlock()

	// Return the data for the agent
	return &api.DeleteEntityResponse{
		Deleted: 1,
	}, nil
}

// Get data for an entity
func (s *environmentServer) ExecuteAgentAction(ctx context.Context, req *api.ExecuteAgentActionRequest) (*api.ExecuteAgentActionResponse, error) {
	// Lock the data, defer unlock until end of call
	s.m.Lock()
	defer s.m.Unlock()

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
