package v1

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	firebase "firebase.google.com/go"

	"github.com/olamai/simulation/pkg/stadium/v1"
	"github.com/olamai/simulation/pkg/vec2/v1"
	"github.com/olamai/simulation/pkg/world/v1"

	v1 "github.com/olamai/simulation/pkg/api/v1"
	"github.com/olamai/simulation/pkg/logger"
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
	// World that handles entities
	world world.World
	// Stadium handles spectators
	stadium stadium.Stadium
	// --- Remote Models ---
	// Map from user id to map from model name to channel
	remoteModelMap map[string][]*remoteModel
	// --- Firebase ---
	// Firebase app
	firebaseApp *firebase.App
	// Mutex to ensure data safety
	m sync.Mutex
}

// NewSimulationServiceServer creates simulation service
func NewSimulationServiceServer(env string) v1.SimulationServiceServer {
	s := &simulationServiceServer{
		env:            env,
		stadium:        stadium.NewStadium(),
		remoteModelMap: make(map[string][]*remoteModel),
		firebaseApp:    initializeFirebaseApp(env),
	}
	s.world = world.NewWorld(regionSize, s.onCellUpdate, true)

	// Remove all remote models that were registered for this server before starting
	removeAllRemoteModelsFromFirebase(s.firebaseApp, s.env)

	// Start the environment agent model stepper
	// [ENV CHECK] - in training we don't use RMs so this is unecessary
	if env != "training" {
		go s.remoteModelStepper()
	}
	return s
}

func (s *simulationServiceServer) onCellUpdate(pos vec2.Vec2, e *world.Entity) {
	s.stadium.BroadcastCellUpdate(pos, regionSize, e)
}

// checkAPI checks if the API version requested by client is supported by server
func (s *simulationServiceServer) checkAPI(api string) error {
	// API version is "" means use current version of the service
	if len(api) > 0 {
		if apiVersion != api {
			return status.Errorf(codes.Unimplemented,
				"unsupported API version: service implements API version '%s', but asked for '%s'", apiVersion, api)
		}
	}
	return nil
}

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

// Get data for an entity
func (s *simulationServiceServer) GetEntity(ctx context.Context, req *v1.GetEntityRequest) (*v1.GetEntityResponse, error) {
	// Lock the data, defer unlock until end of call
	s.m.Lock()
	defer s.m.Unlock()
	// check if the API version requested by client is supported by server
	if err := s.checkAPI(req.Api); err != nil {
		return nil, err
	}
	// Get the entity from the map
	entity := s.world.GetEntity(req.Id)
	// Throw an error if an agent by that id doesn't exist
	if entity == nil {
		err := errors.New("GetEntity(): Entity Not Found")
		return nil, err
	}

	// Return the data for the agent
	return &v1.GetEntityResponse{
		Api: apiVersion,
		Entity: &v1.Entity{
			Id:    entity.ID,
			Class: entity.Class,
		},
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

func (s *simulationServiceServer) ResetWorld(ctx context.Context, req *v1.ResetWorldRequest) (*v1.ResetWorldResponse, error) {
	// Lock the data, defer unlock until end of call
	s.m.Lock()
	defer s.m.Unlock()
	// check if the API version requested by client is supported by server
	if err := s.checkAPI(req.Api); err != nil {
		return nil, err
	}
	// Verify the auth token
	profile, err := authenticateFirebaseAccountWithSecret(ctx, s.firebaseApp, s.env)
	if err != nil {
		return nil, errors.New("ResetWorld(): Unable to verify auth token")
	}
	// Only admins can do this in prod
	// Env check
	if s.env == "prod" {
		if profile["role"].(string) != "admin" {
			return nil, errors.New("ResetWorld(): This function is not available in production")
		}
	}

	// Reset the world
	s.world.Reset()
	// Broadcast the reset
	s.stadium.BroadcastServerAction("RESET")
	// Return
	return &v1.ResetWorldResponse{}, nil
}

// Remove an agent
func (s *simulationServiceServer) CreateSpectator(req *v1.CreateSpectatorRequest, stream v1.SimulationService_CreateSpectatorServer) error {
	// Get spectator ID from client in the request
	spectatorID := req.Id
	// Lock the data, unlock after spectator is added
	s.m.Lock()
	channel := s.stadium.AddSpectator(spectatorID)
	// Unlock data
	s.m.Unlock()

	// Listen for updates and send them to the client
	for {
		response := <-channel
		if err := stream.Send(&response); err != nil {
			// Break the sending loop
			break
		}
	}

	// Remove the spectator and clean up
	// Lock data until spectator is removed
	s.m.Lock()
	s.stadium.RemoveSpectator(spectatorID)
	// Unlock data
	s.m.Unlock()
	log.Printf("Spectator left...")

	return nil
}

// Get an observation for an agent
func (s *simulationServiceServer) SubscribeSpectatorToRegion(ctx context.Context, req *v1.SubscribeSpectatorToRegionRequest) (*v1.SubscribeSpectatorToRegionResponse, error) {
	// customHeader := ctx.Value("custom-header=1")
	id := req.Id
	region := vec2.Vec2{X: req.Region.X, Y: req.Region.Y}

	// Lock the data while creating the spectator
	s.m.Lock()
	// If the user is already subbed, successful is false
	successful := s.stadium.SubscribeSpectatorToRegion(id, region)
	if !successful {
		s.m.Unlock()
		return &v1.SubscribeSpectatorToRegionResponse{
			Api:        apiVersion,
			Successful: false,
		}, nil
	}
	// Get spectator channel
	exists := s.stadium.DoesSpectatorExist(id)
	// Unlock the data
	s.m.Unlock()

	// If the channel hasn't been created yet, try waiting a couple seconds then trying again
	//  Try this 3 times
	for i := 1; i < 4; i++ {
		if exists {
			break
		}
		logger.Log.Warn("SubscribeSpectatorToRegion(): Spectator channel is nil, sleeping and trying again. Try #" + string(i))
		time.Sleep(2 * time.Second)
		// Lock the data when attempting to read from spect map
		s.m.Lock()
		exists = s.stadium.DoesSpectatorExist(id)
		// Unlock the data
		s.m.Unlock()
	}

	// If after the retrys it still hasn't found a channel throw an error
	if !exists {
		return nil, errors.New("SubscribeSpectatorToRegion(): Couldn't find a spectator by that id")
	}

	// Lock the data while sending the spectator the initial region data
	s.m.Lock()
	defer s.m.Unlock()

	// Send initial region state
	positions := region.GetPositionsInRegion(regionSize)
	for _, pos := range positions {
		if entity := s.world.GetEntityByPos(pos); entity != nil {
			s.stadium.SendCellUpdate(id, pos, entity)
		}
	}

	return &v1.SubscribeSpectatorToRegionResponse{
		Api:        apiVersion,
		Successful: true,
	}, nil
}

func (s *simulationServiceServer) UnsubscribeSpectatorFromRegion(ctx context.Context, req *v1.UnsubscribeSpectatorFromRegionRequest) (*v1.UnsubscribeSpectatorFromRegionResponse, error) {
	// customHeader := ctx.Value("custom-header=1")
	id := req.Id
	region := vec2.Vec2{X: req.Region.X, Y: req.Region.Y}

	// Lock the data while creating the spectator
	s.m.Lock()
	defer s.m.Unlock()
	// Attempt to unsub
	successful := s.stadium.UnsubscribeSpectatorFromRegion(id, region)
	if !successful {
		return &v1.UnsubscribeSpectatorFromRegionResponse{
			Api:        apiVersion,
			Successful: false,
		}, nil
	}

	return &v1.UnsubscribeSpectatorFromRegionResponse{
		Api:        apiVersion,
		Successful: true,
	}, nil
}

func (s *simulationServiceServer) CreateRemoteModel(req *v1.CreateRemoteModelRequest, stream v1.SimulationService_CreateRemoteModelServer) error {
	ctx := stream.Context()
	// Check if the API version requested by client is supported by server
	if err := s.checkAPI(req.Api); err != nil {
		return err
	}

	// Lock the data, defer unlock until end of call
	s.m.Lock()

	// Get profile from
	profile, err := authenticateFirebaseAccountWithSecret(ctx, s.firebaseApp, s.env)
	if err != nil {
		// Unlock the data
		s.m.Unlock()
		return err
	}

	// Add a channel for this remote model
	remoteModel, err := s.addRemoteModel(profile["id"].(string), req.Name)
	if err != nil {
		// Unlock the data
		s.m.Unlock()
		return err
	}
	// Unlock the data
	s.m.Unlock()

	// Channel that, when a value is sent to it, will stop this thread and
	//  in turn gracefully remove this RM.
	stopRM := make(chan int)
	// Listen for outgoing messages for the RM and send them
	go func() {
		for {
			v := <-remoteModel.channel
			if err := stream.Send(&v); err != nil {
				stopRM <- 1
			}
		}
	}()
	// Listen for Context Done message
	go func() {
		for {
			<-ctx.Done()
			stopRM <- 1
		}
	}()

	// Wait for the channel to receive a value before stopping
	<-stopRM

	logger.Log.Warn("CreateRemoteModel(): Model has disconnected or timed out")

	// Remove the remote model and clean up
	// Lock data until spectator is removed
	s.m.Lock()
	s.removeRemoteModel(profile["id"].(string), req.Name)
	// Unlock data
	s.m.Unlock()

	return nil
}
