package v1

import (
	"context"
	"sync"

	firebase "firebase.google.com/go"

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
type simulationServiceServer struct {
	// Environment the server is running in
	env string
	// --- Firebase ---
	// Firebase app
	firebaseApp *firebase.App
	// Mutex to ensure data safety
	m sync.Mutex
}

// NewSimulationServiceServer creates simulation service
func NewSimulationServiceServer(env string) v1.SimulationServiceServer {
	s := &simulationServiceServer{
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
func (s *simulationServiceServer) CreateEntity(ctx context.Context, req *v1.CreateEntityRequest) (*v1.CreateEntityResponse, error) {
	// Lock the data, defer unlock until end of call
	s.m.Lock()
	defer s.m.Unlock()
	// check if the API version requested by client is supported by server
	if err := s.checkAPI(req.Api); err != nil {
		return nil, err
	}

	// Return the data for the agent
	return &v1.CreateEntityResponse{
		Api: apiVersion,
		Id:  0,
	}, nil
}

// Get data for an entity
func (s *simulationServiceServer) GetEntity(ctx context.Context, req *v1.GetEntityRequest) (*v1.GetEntityResponse, error) {
	// Lock the data, defer unlock until end of call
	s.m.Lock()
	defer s.m.Unlock()
	// check if the API version requested by client is supported by server
	if err := s.checkAPI(req.Api); err != nil {
		return nil, err
	}

	// Return the data for the agent
	return &v1.GetEntityResponse{
		Api: apiVersion,
		Entity: &v1.Entity{
			Id:    0,
			Class: "AGENT",
		},
	}, nil
}

// Get data for an entity
func (s *simulationServiceServer) DeleteEntity(ctx context.Context, req *v1.DeleteEntityRequest) (*v1.DeleteEntityResponse, error) {
	// Lock the data, defer unlock until end of call
	s.m.Lock()
	defer s.m.Unlock()
	// check if the API version requested by client is supported by server
	if err := s.checkAPI(req.Api); err != nil {
		return nil, err
	}

	// Return the data for the agent
	return &v1.DeleteEntityResponse{
		Api:     apiVersion,
		Deleted: 1,
	}, nil
}

// Get data for an entity
func (s *simulationServiceServer) ExecuteAgentAction(ctx context.Context, req *v1.ExecuteAgentActionRequest) (*v1.ExecuteAgentActionResponse, error) {
	// Lock the data, defer unlock until end of call
	s.m.Lock()
	defer s.m.Unlock()
	// check if the API version requested by client is supported by server
	if err := s.checkAPI(req.Api); err != nil {
		return nil, err
	}

	// Return the data for the agent
	return &v1.ExecuteAgentActionResponse{
		Api:           apiVersion,
		WasSuccessful: true,
	}, nil
}

func (s *simulationServiceServer) ResetWorld(ctx context.Context, req *v1.ResetWorldRequest) (*v1.ResetWorldResponse, error) {
	// Lock the data, defer unlock until end of call
	s.m.Lock()
	defer s.m.Unlock()
	// check if the API version requested by client is supported by server
	if err := s.checkAPI(req.Api); err != nil {
		return nil, err
	}

	// Return
	return &v1.ResetWorldResponse{}, nil
}
