package environment

import (
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"sync"

	"google.golang.org/grpc/metadata"

	uuid "github.com/satori/go.uuid"

	datacom "github.com/terrariumai/simulation/pkg/datacom"

	"github.com/golang/protobuf/ptypes/empty"
	collectiveApi "github.com/terrariumai/simulation/pkg/api/collective"
	envApi "github.com/terrariumai/simulation/pkg/api/environment"
)

const (
	// apiVersion is version of API is provided by server
	apiVersion            = "v1"
	agentLivingEnergyCost = 2
	minFoodBeforeRespawn  = 200
	regionSize            = 16
	maxPositionPadding    = 3
	maxPosition           = 999
	minPosition           = 1

	livingEnergyCost = 1
	moveCost         = 1
	startingEnergy   = 100
	startingHealth   = 100
)

// toDoServiceServer is implementation of api.ToDoServiceServer proto interface
type environmentServer struct {
	// Environment the server is running in
	env string
	// Datacom
	datacomDAL DataAccessLayer
	// Mutex to ensure data safety
	m sync.Mutex
}

// UserInfo is the struct that will parse the auth response
type UserInfo struct {
	Issuer string `json:"issuer"`
	ID     string `json:"id"`
	Email  string `json:"email"`
}

// DataAccessLayer interface for all data access, specificly plugs in from datacom
type DataAccessLayer interface {
	// Redis
	IsCellOccupied(x uint32, y uint32) (bool, error)
	CreateEntity(e envApi.Entity) error
	DeleteEntity(id string) (int64, error)
	UpdateEntity(origionalContent string, e envApi.Entity) error
	GetEntity(id string) (*envApi.Entity, *string, error)
	GetEntitiesForModel(modelID string) ([]envApi.Entity, error)
	GetObservationForEntity(entity envApi.Entity) (*collectiveApi.Observation, error)
	GetEntitiesInRegion(x uint32, y uint32) ([]*envApi.Entity, error)
	// Firebase
	GetRemoteModelMetadataBySecret(modelSecret string) (*datacom.RemoteModel, error)
	GetRemoteModelMetadataByID(modelID string) (*datacom.RemoteModel, error)
	UpdateRemoteModelMetadata(remoteModelMD *datacom.RemoteModel, connectCount int) error
}

// NewEnvironmentServer creates simulation service
func NewEnvironmentServer(env string, d DataAccessLayer) envApi.EnvironmentServer {
	// initialize server
	s := &environmentServer{
		env:        env,
		datacomDAL: d,
	}

	return s
}

// Get data for an entity
func (s *environmentServer) CreateEntity(ctx context.Context, req *envApi.CreateEntityRequest) (*envApi.CreateEntityResponse, error) {
	// Lock the data, defer unlock until end of call
	s.m.Lock()
	defer s.m.Unlock()

	// Get user info from metadata
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		err := errors.New("Incorrect or no headers were provided")
		log.Printf("ERROR: %s\n", err)
		return nil, err
	}
	userInfoHeader := md["x-endpoint-api-userinfo"]
	sDec, _ := b64.StdEncoding.DecodeString(userInfoHeader[0])
	userInfo := UserInfo{}
	json.Unmarshal(sDec, &userInfo)

	// Make sure the user has supplied data
	if req.Entity == nil {
		err := errors.New("Entity not in request")
		log.Printf("%v", err)
		return nil, err
	}

	// Validate entity class
	if req.Entity.Class < 0 || req.Entity.Class > 3 {
		err := errors.New("Error: invalid class")
		log.Printf("Error: %v\n", err)
		return nil, err
	}

	// Validate modelID
	if len(req.Entity.ModelID) == 0 {
		err := errors.New("Error: missing model id")
		log.Printf("Error: %v\n", err)
		return nil, err
	}
	remoteModelMD, err := s.datacomDAL.GetRemoteModelMetadataByID(req.Entity.ModelID)
	if err != nil {
		log.Printf("%v\n", err)
		return nil, err
	}
	if remoteModelMD.OwnerUID != userInfo.ID {
		err := errors.New("you do not own that remote model")
		log.Printf("Error validating modelID. Metadata owner=%s, but userinfo id=%s\n", remoteModelMD.OwnerUID, userInfo.ID)
		return nil, err
	}
	if remoteModelMD.ConnectCount == 0 {
		err := errors.New("you must connect your remote model before creating entities for it")
		log.Printf("Error validating modelID: %v\n", err)
		return nil, err
	}

	// Make sure the cell is not occupied
	isCellOccupied, err := s.datacomDAL.IsCellOccupied(req.Entity.X, req.Entity.Y)
	if err != nil {
		log.Printf("Error checking if cell is occupied")
		return nil, err
	}
	if isCellOccupied {
		log.Printf("Error cell is occupied")
		return nil, errors.New("That cell is already occupied by an entity")
	}

	// Create an id for the entity
	newUUID, err := uuid.NewV4()
	if err != nil {
		err := errors.New("Error generating id")
		log.Printf("ERROR CreateEntity(): %v\n", err)
		return nil, err
	}
	entityID := newUUID.String()
	// Or... use given ID for testing
	if s.env == "testing" {
		entityID = req.Entity.Id
	}

	// Set values for the entity
	req.Entity.OwnerUID = userInfo.ID
	req.Entity.Energy = startingEnergy
	req.Entity.Health = startingHealth
	req.Entity.Id = entityID

	// Add the entity to the environment
	err = s.datacomDAL.CreateEntity(*req.Entity)

	// Return the data for the agent
	return &envApi.CreateEntityResponse{
		Id: entityID,
	}, nil
}

// Get data for an entity
func (s *environmentServer) GetEntity(ctx context.Context, req *envApi.GetEntityRequest) (*envApi.GetEntityResponse, error) {
	// Lock the data, defer unlock until end of call
	s.m.Lock()
	defer s.m.Unlock()

	// Get the entity
	entity, _, err := s.datacomDAL.GetEntity(req.Id)
	if err != nil {
		return nil, errors.New("Couldn't find an entity by that id")
	}

	// Return the data for the agent
	return &envApi.GetEntityResponse{
		Entity: entity,
	}, nil
}

// Get data for an entity
func (s *environmentServer) DeleteEntity(ctx context.Context, req *envApi.DeleteEntityRequest) (*envApi.DeleteEntityResponse, error) {
	// Lock the data, defer unlock until end of call
	s.m.Lock()
	defer s.m.Unlock()

	// Remove the entity from the environment
	deleted, err := s.datacomDAL.DeleteEntity(req.Id)
	if err != nil {
		return nil, err
	}

	// Return the data for the agent
	return &envApi.DeleteEntityResponse{
		Deleted: deleted,
	}, nil
}

// Get data for an entity
func (s *environmentServer) ExecuteAgentAction(ctx context.Context, req *envApi.ExecuteAgentActionRequest) (*envApi.ExecuteAgentActionResponse, error) {
	// Lock the data, defer unlock until end of call
	s.m.Lock()
	defer s.m.Unlock()

	fmt.Printf("Execute agent action: %v \n", req)
	// Get the entity
	entity, origionalContent, err := s.datacomDAL.GetEntity(req.Id)
	if err != nil {
		return nil, err
	}

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

	// Living energy cost
	// Note: we will handle negative energy as overflow later
	if entity.Energy > 0 {
		entity.Energy -= livingEnergyCost
	} else {
		entity.Health -= livingEnergyCost
	}

	switch req.Action {
	case 0: // REST
	case 1: // MOVE
		if targetX < minPosition || targetY < minPosition {
			return &envApi.ExecuteAgentActionResponse{
				WasSuccessful: false,
			}, nil
		}
		// Check if cell is occupied
		isCellOccupied, err := s.datacomDAL.IsCellOccupied(targetX, targetY)
		if isCellOccupied || err != nil {
			// Return unsuccessful
			return &envApi.ExecuteAgentActionResponse{
				WasSuccessful: false,
			}, nil
		}
		entity.X = targetX
		entity.Y = targetY
	default: // INVALID
		return &envApi.ExecuteAgentActionResponse{
			WasSuccessful: false,
		}, nil
	}

	// Handle overflow energy
	if entity.Energy < 0 {
		overflow := uint32(math.Abs(float64(entity.Energy)))
		entity.Health -= overflow
	}

	// Handle death case
	if entity.Health <= 0 {
		s.datacomDAL.DeleteEntity(entity.Id)
		return &envApi.ExecuteAgentActionResponse{
			WasSuccessful: false,
		}, nil
	}

	// Update the entity
	err = s.datacomDAL.UpdateEntity(*origionalContent, *entity)
	if err != nil {
		return nil, err
	}

	// Return the data for the agent
	return &envApi.ExecuteAgentActionResponse{
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

func (s *environmentServer) GetEntitiesInRegion(ctx context.Context, req *envApi.GetEntitiesInRegionRequest) (*envApi.GetEntitiesInRegionResponse, error) {
	// Lock the data, defer unlock until end of call
	s.m.Lock()
	defer s.m.Unlock()
	entities := []*envApi.Entity{}

	entities, err := s.datacomDAL.GetEntitiesInRegion(req.X, req.Y)
	if err != nil {
		log.Printf("GetEntitiesInRegion(): error %v", err)
	}

	return &envApi.GetEntitiesInRegionResponse{
		Entities: entities,
	}, nil
}
