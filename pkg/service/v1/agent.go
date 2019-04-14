package v1

import (
	"context"
	"errors"

	v1 "github.com/olamai/simulation/pkg/api/v1"
	"github.com/olamai/simulation/pkg/vec2/v1"
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

	// Create a new agent
	if s.env == "prod" {
		if !s.doesRemoteModelExist(profile["id"].(string), req.ModelName) {
			return nil, errors.New("CreateNewEntity(): That model does not exist")
		}
	}
	agent, err := s.world.NewAgentEntity(profile["id"].(string), req.ModelName, vec2.Vec2{X: req.X, Y: req.Y})
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
	// Get data from request
	action := req.Action
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
			Api:                 apiVersion,
			IsAgentStillAlive:   false,
			WasActionSuccessful: false,
		}, nil
	}

	// Get the target position from the given direction and agent
	targetPos, err := getTargetPosFromDirectionAndAgent(action.Direction, agent)
	if err != nil {
		return nil, err
	}

	// Perform the action
	switch action.Id {
	case "MOVE":
		actionSuccess = s.world.EntityMove(agent.ID, targetPos)
	case "CONSUME":
		actionSuccess = s.world.EntityConsume(agent.ID, targetPos)
	}

	// Only subtract living cost on actions during training, otherwise do it
	//   in the RM stepper
	if s.env == "training" {
		s.world.EntityLivingCostUpdate(agent)
	}

	// If the agent died during all this, return that
	if !s.world.DoesEntityExist(agent.ID) {
		return &v1.ExecuteAgentActionResponse{
			Api:                 apiVersion,
			IsAgentStillAlive:   false,
			WasActionSuccessful: false,
		}, nil
	}

	// Agent is still alive
	return &v1.ExecuteAgentActionResponse{
		Api:                 apiVersion,
		IsAgentStillAlive:   true,
		WasActionSuccessful: actionSuccess,
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

	if e != nil {
		cells := s.world.GetObservationCellsForPosition(e.Pos)
		// Agent is alive and well... maybe, at least it's alive
		return &v1.GetAgentObservationResponse{
			Api: apiVersion,
			Observation: &v1.Observation{
				Id:     e.ID,
				Alive:  true,
				Cells:  cells,
				Energy: e.Energy,
				Health: e.Health,
			},
		}, nil
	}
	// Agent doesn't exist anymore
	return &v1.GetAgentObservationResponse{
		Api: apiVersion,
		Observation: &v1.Observation{
			Id:     0,
			Alive:  false,
			Cells:  []string{},
			Energy: 0,
			Health: 0,
		},
	}, nil
}
