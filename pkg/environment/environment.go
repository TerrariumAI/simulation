package environment

import (
	"context"
	"fmt"
	"os"
	"sync"

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

	err = s.redisClient.Set("key", user["id"].(string), 0).Err()
	if err != nil {
		panic(err)
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
