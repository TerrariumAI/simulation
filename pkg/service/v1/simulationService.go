package v1

import (
	"context"
	"sync"

	firebase "firebase.google.com/go"

	"github.com/golang/protobuf/ptypes/empty"
	v1 "github.com/terrariumai/simulation/pkg/api/v1"
	"github.com/terrariumai/simulation/pkg/logger"
)

const (
	// apiVersion is version of API is provided by server
	apiVersion            = "v1"
	agentLivingEnergyCost = 2
	minFoodBeforeRespawn  = 200
	regionSize            = 16
)

// toDoServiceServer is implementation of v1.ToDoServiceServer proto interface
type simulationServer struct {
	// Environment the server is running in
	env string
	// --- Firebase ---
	// Firebase app
	firebaseApp *firebase.App
	// Mutex to ensure data safety
	m sync.Mutex
}

// NewSimulationServiceServer creates simulation service
func NewSimulationServiceServer(env string) v1.SimulationServer {
	s := &simulationServer{
		env:         env,
		firebaseApp: initializeFirebaseApp(env),
	}

	if env == "testing" {
		logger.Init(-1, "")
	}

	// // Remove all remote models that were registered for this server before starting
	// removeAllRemoteModelsFromFirebase(s.firebaseApp, s.env)

	return s
}

// Get data for an entity
func (s *simulationServer) CreateEntity(ctx context.Context, req *v1.CreateEntityRequest) (*v1.CreateEntityResponse, error) {
	// Lock the data, defer unlock until end of call
	s.m.Lock()
	defer s.m.Unlock()

	// Return the data for the agent
	return &v1.CreateEntityResponse{
		Id: 0,
	}, nil
}

// Get data for an entity
func (s *simulationServer) GetEntity(ctx context.Context, req *v1.GetEntityRequest) (*v1.GetEntityResponse, error) {
	// Lock the data, defer unlock until end of call
	s.m.Lock()
	defer s.m.Unlock()

	// Return the data for the agent
	return &v1.GetEntityResponse{
		Entity: &v1.Entity{
			Id:    0,
			Class: "AGENT",
		},
	}, nil
}

// Get data for an entity
func (s *simulationServer) DeleteEntity(ctx context.Context, req *v1.DeleteEntityRequest) (*v1.DeleteEntityResponse, error) {
	// Lock the data, defer unlock until end of call
	s.m.Lock()
	defer s.m.Unlock()

	// Return the data for the agent
	return &v1.DeleteEntityResponse{
		Deleted: 1,
	}, nil
}

// Get data for an entity
func (s *simulationServer) ExecuteAgentAction(ctx context.Context, req *v1.ExecuteAgentActionRequest) (*v1.ExecuteAgentActionResponse, error) {
	// Lock the data, defer unlock until end of call
	s.m.Lock()
	defer s.m.Unlock()

	// Return the data for the agent
	return &v1.ExecuteAgentActionResponse{
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
