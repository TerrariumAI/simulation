package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	firebase "firebase.google.com/go"

	"github.com/olamai/simulation/pkg/stadium/v1"
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

// Web which stores google ids.
type Web struct {
	ClientID     string `json:"client_id"`
	ProjectID    string `json:"project_id"`
	AuthURI      string `json:"auth_uri"`
	TokenURI     string `json:"token_uri"`
	ClientSecret string `json:"client_secret"`
}

// OAuthCredentials which stores google ids.
type OAuthCredentials struct {
	Web Web `json:"web"`
}

// NewSimulationServiceServer creates simulation service
func NewSimulationServiceServer(env string) v1.SimulationServiceServer {
	s := &simulationServiceServer{
		env:            env,
		stadium:        stadium.NewStadium(regionSize),
		remoteModelMap: make(map[string][]*remoteModel),
		firebaseApp:    initializeFirebaseApp(env),
	}
	s.world = world.NewWorld(regionSize, s.stadium.BroadcastCellUpdate, true)

	// Remove all remote models that were registered for this server before starting
	removeAllRemoteModelsFromFirebase(s.firebaseApp, s.env)

	// Start the environment agent model stepper
	// [ENV CHECK] - in training we don't use RMs so this is unecessary
	go s.stepWorldContinuous()

	// ----------------
	// -- OAUTH TESTING
	// ----------------
	var c OAuthCredentials
	file, err := ioutil.ReadFile("./oauthCreds.json")
	if err != nil {
		fmt.Printf("File error: %v\n", err)
		os.Exit(1)
	}
	json.Unmarshal(file, &c)
	println(c.Web.ClientID)

	conf := &oauth2.Config{
		ClientID:     c.Web.ClientID,
		ClientSecret: c.Web.ClientSecret,
		RedirectURL:  "http://localhost:3000/auth",
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email", // You have to select your own scope from here -> https://developers.google.com/identity/protocols/googlescopes#google_sign-in
		},
		Endpoint: google.Endpoint,
	}

	return s
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

func (s *simulationServiceServer) StepWorld(ctx context.Context, req *v1.StepWorldRequest) (*v1.StepWorldResponse, error) {
	// Lock the data, defer unlock until end of call
	s.m.Lock()
	defer s.m.Unlock()
	// check if the API version requested by client is supported by server
	if err := s.checkAPI(req.Api); err != nil {
		return nil, err
	}
	// only available when not in prod
	if s.env == "prod" {
		return nil, errors.New("StepWorld(): This function is not available")
	}

	// Reset the world
	s.stepWorldOnce()
	// Return
	return &v1.StepWorldResponse{}, nil
}
