package simulation

import (
	"context"
	"sync"

	firebase "firebase.google.com/go"

	"github.com/golang/protobuf/ptypes/empty"
	api "github.com/terrariumai/simulation/pkg/api"
)

const (
	// apiVersion is version of API is provided by server
	apiVersion            = "v1"
	agentLivingEnergyCost = 2
	minFoodBeforeRespawn  = 200
	regionSize            = 16
)

// toDoServiceServer is implementation of api.ToDoServiceServer proto interface
type simulationServer struct {
	// Environment the server is running in
	env string
	// --- Firebase ---
	// Firebase app
	firebaseApp *firebase.App
	// Mutex to ensure data safety
	m sync.Mutex
}

// NewSimulationServer creates simulation service
func NewSimulationServer(env string) api.SimulationServer {
	s := &simulationServer{
		env:         env,
		firebaseApp: initializeFirebaseApp(env),
	}

	// // Remove all remote models that were registered for this server before starting
	// removeAllRemoteModelsFromFirebase(s.firebaseApp, s.env)

	return s
}

// Get data for an entity
func (s *simulationServer) CreateEntity(ctx context.Context, req *api.CreateEntityRequest) (*api.CreateEntityResponse, error) {
	// Lock the data, defer unlock until end of call
	s.m.Lock()
	defer s.m.Unlock()

	// Return the data for the agent
	return &api.CreateEntityResponse{
		Id: 0,
	}, nil
}

// Get data for an entity
func (s *simulationServer) GetEntity(ctx context.Context, req *api.GetEntityRequest) (*api.GetEntityResponse, error) {
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
func (s *simulationServer) DeleteEntity(ctx context.Context, req *api.DeleteEntityRequest) (*api.DeleteEntityResponse, error) {
	// Lock the data, defer unlock until end of call
	s.m.Lock()
	defer s.m.Unlock()

	// Return the data for the agent
	return &api.DeleteEntityResponse{
		Deleted: 1,
	}, nil
}

// Get data for an entity
func (s *simulationServer) ExecuteAgentAction(ctx context.Context, req *api.ExecuteAgentActionRequest) (*api.ExecuteAgentActionResponse, error) {
	// Lock the data, defer unlock until end of call
	s.m.Lock()
	defer s.m.Unlock()

	// Return the data for the agent
	return &api.ExecuteAgentActionResponse{
		WasSuccessful: true,
	}, nil
}

func (s *simulationServer) ResetWorld(ctx context.Context, req *empty.Empty) (*empty.Empty, error) {
	// Lock the data, defer unlock until end of call
	s.m.Lock()
	defer s.m.Unlock()

	// Return
	return &empty.Empty{}, nil
}
