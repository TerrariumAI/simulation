package environment

import (
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sync"

	"google.golang.org/grpc/metadata"

	uuid "github.com/satori/go.uuid"

	datacom "github.com/terrariumai/simulation/pkg/datacom"

	"github.com/golang/protobuf/ptypes/empty"
	collectiveApi "github.com/terrariumai/simulation/pkg/api/collective"
	envApi "github.com/terrariumai/simulation/pkg/api/environment"
)

const (
	maxPosition            = 999
	minPosition            = 1
	maxUserCreatedEntities = 5

	agentLivingEnergyCost = 1
	agentMoveEnergyCost   = 2
	agentAttackEnergyCost = 5
	agentEnergyGainOnEat  = 10
	agentAttackDmg        = 10
	startingEnergy        = 100
	startingHealth        = 100
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
	IsCellOccupied(x uint32, y uint32) (bool, *envApi.Entity, string, error)
	CreateEntity(e envApi.Entity, shouldPublish bool) error
	DeleteEntity(id string) (int64, error)
	UpdateEntity(origionalContent string, e envApi.Entity) error
	GetEntity(id string) (*envApi.Entity, string, error)
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
		log.Printf("ERROR: %v\n", err)
		return nil, err
	}
	userInfoHeader := md["x-endpoint-api-userinfo"]
	sDec, _ := b64.StdEncoding.DecodeString(userInfoHeader[0])
	userInfo := UserInfo{}
	json.Unmarshal(sDec, &userInfo)

	// Make sure the user has supplied data
	if req.Entity == nil {
		err := errors.New("entity not in request")
		log.Printf("ERROR: %v\n", err)
		return nil, err
	}

	// Validate entity class
	if req.Entity.ClassID > 3 {
		err := errors.New("invalid class")
		log.Printf("ERROR: %v\n", err)
		return nil, err
	}

	// Validate modelID
	if len(req.Entity.ModelID) == 0 {
		err := errors.New("missing model id")
		log.Printf("ERROR: %v\n", err)
		return nil, err
	}
	remoteModelMD, err := s.datacomDAL.GetRemoteModelMetadataByID(req.Entity.ModelID)
	if err != nil {
		log.Printf("ERROR: %v\n", err)
		return nil, err
	}
	if remoteModelMD.OwnerUID != userInfo.ID {
		err := errors.New("you do not own that remote model")
		log.Printf("Error validating modelID. Metadata owner=%s, but userinfo id=%s\n", remoteModelMD.OwnerUID, userInfo.ID)
		return nil, err
	}
	if remoteModelMD.ConnectCount == 0 {
		err := errors.New("rm is offline")
		log.Printf("ERROR: %v\n", err)
		return nil, err
	}

	// Make sure user can't create more than limit
	entities, err := s.datacomDAL.GetEntitiesForModel(remoteModelMD.ID)
	if err != nil {
		log.Printf("ERROR querying entities: %v\n", err)
		return nil, err
	}
	if len(entities) >= maxUserCreatedEntities && s.env != "training" {
		err := errors.New("you can only manually create 5 entities at a time")
		return nil, err
	}

	// Validate position
	if req.Entity.X < minPosition || req.Entity.Y < minPosition {
		err := errors.New("invalid position")
		log.Printf("ERROR: %v\n", err)
		return nil, err
	}
	if req.Entity.X > maxPosition || req.Entity.Y > maxPosition {
		err := errors.New("invalid position")
		log.Printf("ERROR: %v\n", err)
		return nil, err
	}

	// Make sure the cell is not occupied
	isCellOccupied, _, _, err := s.datacomDAL.IsCellOccupied(req.Entity.X, req.Entity.Y)
	if err != nil {
		log.Printf("ERROR: %v\n", err)
		return nil, err
	}
	if isCellOccupied {
		err := errors.New("cell is already occupied")
		log.Printf("ERROR: %v\n", err)
		return nil, err
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
	err = s.datacomDAL.CreateEntity(*req.Entity, true)

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
		fmt.Printf("ERROR: %v\n", err)
		return nil, err
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

	// Get the entity
	entity, origionalContent, err := s.datacomDAL.GetEntity(req.Id)
	if err != nil {
		// Note: returning an error here seems correct, but completely stops a model if an agent was deleted
		// mid session.
		// An error here essentially means the agent was removed manually.
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
	if entity.Energy >= agentLivingEnergyCost {
		entity.Energy -= agentLivingEnergyCost
	} else {
		diff := agentLivingEnergyCost - entity.Energy
		if entity.Health >= diff {
			entity.Health -= diff
		} else {
			// KILL
			s.datacomDAL.DeleteEntity(entity.Id)
			return &envApi.ExecuteAgentActionResponse{
				WasSuccessful: false,
				IsAlive:       false,
			}, nil
		}
	}

	switch req.Action {
	case 0: // REST
	case 1: // MOVE
		if targetX < minPosition || targetY < minPosition {
			return &envApi.ExecuteAgentActionResponse{
				WasSuccessful: false,
				IsAlive:       true,
			}, nil
		}
		// Check if cell is occupied
		isCellOccupied, _, _, err := s.datacomDAL.IsCellOccupied(targetX, targetY)
		if isCellOccupied || err != nil {
			// Return unsuccessful
			return &envApi.ExecuteAgentActionResponse{
				WasSuccessful: false,
				IsAlive:       true,
			}, nil
		}
		entity.X = targetX
		entity.Y = targetY
		// Adjust energy
		if entity.Energy >= agentMoveEnergyCost {
			entity.Energy -= agentMoveEnergyCost
		} else {
			diff := agentMoveEnergyCost - entity.Energy
			if entity.Health >= diff {
				entity.Health -= diff
			} else {
				// KILL
				s.datacomDAL.DeleteEntity(entity.Id)
				return &envApi.ExecuteAgentActionResponse{
					WasSuccessful: false,
					IsAlive:       false,
				}, nil
			}
		}
	case 2: // EAT
		// Check if cell is occupied
		isCellOccupied, other, _, err := s.datacomDAL.IsCellOccupied(targetX, targetY)
		if !isCellOccupied || err != nil {
			// Return unsuccessful
			return &envApi.ExecuteAgentActionResponse{
				WasSuccessful: false,
				IsAlive:       true,
			}, nil
		}
		if other.ClassID != 3 { // FOOD
			return &envApi.ExecuteAgentActionResponse{
				WasSuccessful: false,
				IsAlive:       true,
			}, nil
		}
		// Update entity
		entity.Energy += agentEnergyGainOnEat
		// Delete food
		s.datacomDAL.DeleteEntity(other.Id)
		// Spawn another random food entity (loop to ensure another is spawned)
		// this artificially keeps the ecosystem in check
		for {
			// Create an id for the entity
			newUUID, err := uuid.NewV4()
			if err != nil {
				err := errors.New("Error generating id")
				log.Printf("ERROR CreateEntity(): %v\n", err)
				return nil, err
			}
			entityID := newUUID.String()
			// Create entity
			e := envApi.Entity{
				Id:      entityID,
				ClassID: 3,
				X:       uint32(rand.Intn(100)),
				Y:       uint32(rand.Intn(100)),
			}
			// Create entity silently (no publish)
			err = s.datacomDAL.CreateEntity(e, true)
			if err == nil {
				break
			}
		}
	case 3: // ATTACK
		// Check if cell is occupied
		isCellOccupied, other, otherOrigionalContent, err := s.datacomDAL.IsCellOccupied(targetX, targetY)
		if !isCellOccupied || err != nil {
			// Return unsuccessful
			return &envApi.ExecuteAgentActionResponse{
				WasSuccessful: false,
				IsAlive:       true,
			}, nil
		}
		// Make sure the other entity is an agent
		if other.ClassID != 1 { // AGENT
			return &envApi.ExecuteAgentActionResponse{
				WasSuccessful: false,
				IsAlive:       true,
			}, nil
		}
		// Update this entity's energy
		if entity.Energy >= agentAttackEnergyCost {
			entity.Energy -= agentAttackEnergyCost
		} else {
			diff := agentAttackEnergyCost - entity.Energy
			if entity.Health >= diff {
				entity.Health -= diff
			} else {
				// KILL
				s.datacomDAL.DeleteEntity(entity.Id)
				return &envApi.ExecuteAgentActionResponse{
					WasSuccessful: false,
					IsAlive:       false,
				}, nil
			}
		}
		// Update other entity's health
		if other.Health > agentAttackDmg {
			other.Health -= agentAttackDmg
			// Update the entity
			err = s.datacomDAL.UpdateEntity(otherOrigionalContent, *other)
			if err != nil {
				fmt.Printf("ERROR: %v\n", err)
			}
		} else {
			// KILL
			s.datacomDAL.DeleteEntity(other.Id)
		}

	default: // INVALID
		return &envApi.ExecuteAgentActionResponse{
			WasSuccessful: false,
			IsAlive:       true,
		}, nil
	}

	// Update the entity
	err = s.datacomDAL.UpdateEntity(origionalContent, *entity)
	if err != nil {
		return nil, err
	}

	// Return the data for the agent
	return &envApi.ExecuteAgentActionResponse{
		WasSuccessful: true,
		IsAlive:       true,
	}, nil
}

func (s *environmentServer) ResetWorld(ctx context.Context, req *empty.Empty) (*empty.Empty, error) {
	// Lock the data, defer unlock until end of call
	s.m.Lock()
	defer s.m.Unlock()

	// Return
	return &empty.Empty{}, nil
}

func (s *environmentServer) SpawnFood(ctx context.Context, req *empty.Empty) (*empty.Empty, error) {
	// Lock the data, defer unlock until end of call
	s.m.Lock()
	defer s.m.Unlock()

	// Get user info from metadata
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		err := errors.New("Incorrect or no headers were provided")
		log.Printf("ERROR: %v\n", err)
		return nil, err
	}
	userInfoHeader := md["x-endpoint-api-userinfo"]
	sDec, _ := b64.StdEncoding.DecodeString(userInfoHeader[0])
	userInfo := UserInfo{}
	json.Unmarshal(sDec, &userInfo)
	if userInfo.Email != "zacharyholland@gmail.com" {
		err := errors.New("must be zac to perform this action")
		log.Printf("ERROR: %v\n", err)
		return nil, err
	}

	foodCount := 500
	for i := 0; i < foodCount; i++ {
		// Create an id for the entity
		newUUID, err := uuid.NewV4()
		if err != nil {
			err := errors.New("Error generating id")
			log.Printf("ERROR CreateEntity(): %v\n", err)
			return nil, err
		}
		entityID := newUUID.String()
		// Create entity
		e := envApi.Entity{
			Id:      entityID,
			ClassID: 3,
			X:       uint32(rand.Intn(100)),
			Y:       uint32(rand.Intn(100)),
		}
		// Create entity silently (no publish)
		s.datacomDAL.CreateEntity(e, false)
	}

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
		log.Printf("ERROR: %v", err)
		return nil, err
	}

	return &envApi.GetEntitiesInRegionResponse{
		Entities: entities,
	}, nil
}
