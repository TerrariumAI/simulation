package v1

import (
	"context"
	"errors"

	v1 "github.com/terrariumai/simulation/pkg/api/v1"
	"github.com/terrariumai/simulation/pkg/vec2/v1"
)

// Create new agent
func (s *simulationServiceServer) CreateAgent(ctx context.Context, req *v1.CreateAgentRequest) (*v1.CreateAgentResponse, error) {
	// Lock the data, defer unlock until end of call
	s.m.Lock()
	defer s.m.Unlock()
	// Check if the API version requested by client is supported by server
	if err := s.checkAPI(req.Api); err != nil {
		return nil, err
	}

	// Verify the auth secret
	profile, err := authenticateFirebaseAccountWithSecret(ctx, s.firebaseApp, s.env)
	if err != nil {
		return nil, errors.New("CreateAgent(): Unable to verify auth token")
	}

	// Check for a connected RM ONLY if we are in prod
	if s.env == "prod" {
		if !s.doesRemoteModelExist(profile["id"].(string), req.Agent.ModelName) {
			return nil, errors.New("CreateNewEntity(): That model does not exist")
		}
	}

	// Create the agent
	agent, err := s.world.NewAgentEntity(profile["id"].(string), req.Agent.ModelName, vec2.Vec2{X: req.Agent.Pos.X, Y: req.Agent.Pos.Y})
	if err != nil {
		return nil, err
	}

	return &v1.CreateAgentResponse{
		Api: apiVersion,
		Id:  agent.ID,
	}, nil
}

// Remove an agent
// REQUIRES SECRET KEY FOR AUTH METADATA
// NOT PROD
func (s *simulationServiceServer) DeleteAgent(ctx context.Context, req *v1.DeleteAgentRequest) (*v1.DeleteAgentResponse, error) {
	// Lock the data, defer unlock until end of call
	s.m.Lock()
	defer s.m.Unlock()
	// check if the API version requested by client is supported by server
	if err := s.checkAPI(req.Api); err != nil {
		return nil, err
	}
	// Env check
	if s.env == "prod" {
		return nil, errors.New("DeleteAgent(): This function is not available in production")
	}
	// Verify the auth secret
	_, err := authenticateFirebaseAccountWithSecret(ctx, s.firebaseApp, s.env)
	if err != nil {
		return nil, err
	}

	// Get the agent
	agent := s.world.GetEntity(req.Id)
	// Throw an error if an agent by that id doesn't exist
	if agent == nil {
		err := errors.New("GetAgent(): Agent Not Found")
		return nil, err
	}

	// Remove the entity
	s.world.DeleteEntity(agent.ID)

	// Return the data for the agent
	return &v1.DeleteAgentResponse{
		Api:     apiVersion,
		Deleted: 1,
	}, nil
}

// Execute an action for an agent
func (s *simulationServiceServer) ExecuteAgentAction(ctx context.Context, req *v1.ExecuteAgentActionRequest) (*v1.ExecuteAgentActionResponse, error) {
	// Lock the data, defer unlock until end of call
	s.m.Lock()
	defer s.m.Unlock()
	// Success status to be set later
	var actionSuccess bool
	// check if the API version requested by client is supported by server
	if err := s.checkAPI(req.Api); err != nil {
		return nil, err
	}
	// Get the agent
	agent := s.world.GetEntity(req.Id)
	// Throw an error if an agent by that id doesn't exist
	if agent == nil {
		return &v1.ExecuteAgentActionResponse{
			Api:           apiVersion,
			WasSuccessful: false,
		}, nil
	}

	// Get the target position from the given direction and agent
	targetPos, err := getTargetPosFromDirectionAndAgent(req.Direction, agent)
	if err != nil {
		return nil, err
	}

	// Perform the action
	switch req.Action {
	case 0: // MOVE
		actionSuccess = s.world.EntityMove(agent.ID, targetPos)
	case 1: // CONSUME
		actionSuccess = s.world.EntityConsume(agent.ID, targetPos)
	}

	// If the agent died during all this, return that
	if !s.world.DoesEntityExist(agent.ID) {
		return &v1.ExecuteAgentActionResponse{
			Api:           apiVersion,
			WasSuccessful: false,
		}, nil
	}

	// Agent is still alive
	return &v1.ExecuteAgentActionResponse{
		Api:           apiVersion,
		WasSuccessful: actionSuccess,
	}, nil
}

// Get an observation for an agent
func (s *simulationServiceServer) GetAgentObservation(ctx context.Context, req *v1.GetAgentObservationRequest) (*v1.GetAgentObservationResponse, error) {
	// Lock the data, defer unlock until end of call
	s.m.Lock()
	defer s.m.Unlock()
	// Get the agent
	e := s.world.GetEntity(req.Id)
	// Env check
	if s.env == "prod" {
		return nil, errors.New("GetAgentObservation(): This function is not available in production. Agent observations are sent directly to Remote Models")
	}

	if e == nil {
		// Agent doesn't exist anymore
		return &v1.GetAgentObservationResponse{
			Api: apiVersion,
			Observation: &v1.Observation{
				IsAlive: false,
			},
		}, nil
	}

	cells := s.world.GetObservationCellsForPosition(e.Pos)
	// Agent is alive and well... maybe, at least it's alive
	return &v1.GetAgentObservationResponse{
		Api: apiVersion,
		Observation: &v1.Observation{
			IsAlive: true,
			Entity: &v1.Entity{
				Id:    e.ID,
				Class: e.Class,
				Pos: &v1.Vec2{
					X: e.Pos.X,
					Y: e.Pos.Y,
				},
				Energy:    e.Energy,
				Health:    e.Health,
				OwnerUID:  e.OwnerUID,
				ModelName: e.ModelName,
			},
			Cells: cells,
		},
	}, nil

}
