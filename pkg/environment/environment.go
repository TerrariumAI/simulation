package environment

import (
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"google.golang.org/grpc/metadata"

	uuid "github.com/satori/go.uuid"

	datacom "github.com/terrariumai/simulation/pkg/datacom"

	"github.com/golang/protobuf/ptypes/empty"
	collectiveApi "github.com/terrariumai/simulation/pkg/api/collective"
	envApi "github.com/terrariumai/simulation/pkg/api/environment"
)

const (
	maxPosition            = 100
	minPosition            = 0
	regionSize             = 10
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
	CreateEffect(envApi.Effect) error
	DeleteEntity(id string) (int64, error)
	UpdateEntity(origionalContent string, e envApi.Entity) error
	GetEntity(id string) (*envApi.Entity, string, error)
	GetEntitiesForModel(modelID string) ([]envApi.Entity, error)
	GetObservationForEntity(entity envApi.Entity) (*collectiveApi.Observation, error)
	GetEntitiesInSpace(x0 uint32, y0 uint32, x1 uint32, y1 uint32) ([]*envApi.Entity, error)
	GetEffectsInSpace(x0 uint32, y0 uint32, x1 uint32, y1 uint32) ([]*envApi.Effect, error)
	// Firebase
	GetRemoteModelMetadataBySecret(modelSecret string) (*datacom.RemoteModel, error)
	GetRemoteModelMetadataByID(modelID string) (*datacom.RemoteModel, error)
	UpdateRemoteModelMetadata(remoteModelMD *datacom.RemoteModel, connectCount int) error
	AddEntityMetadataToFireabase(envApi.Entity) error
	RemoveEntityMetadataFromFirebase(id string) error
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

func randomPosition() (uint32, uint32) {
	return uint32(rand.Intn(maxPosition)), uint32(rand.Intn(maxPosition))
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

	// If invalid posiiton, create new random position in the range
	if req.Entity.X < minPosition || req.Entity.Y < minPosition || req.Entity.X > maxPosition || req.Entity.Y > maxPosition {
		x, y := randomPosition()
		req.Entity.X = x
		req.Entity.Y = y
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
	// Add metadata to firebase
	err = s.datacomDAL.AddEntityMetadataToFireabase(*req.Entity)

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
		log.Printf("ERROR: %v", err)
		return nil, err
	}
	// Remove entity from firebase
	err = s.datacomDAL.RemoveEntityMetadataFromFirebase(req.Id)
	if err != nil {
		log.Printf("ERROR: %v", err)
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
			s.datacomDAL.RemoveEntityMetadataFromFirebase(entity.Id)
			return &envApi.ExecuteAgentActionResponse{
				Value: envApi.ExecuteAgentActionResponse_ERR_DIED,
			}, nil
		}
	}

	var targetX, targetY = entity.X, entity.Y
	switch req.Direction {
	case envApi.ExecuteAgentActionRequest_UP: // UP
		targetY++
	case envApi.ExecuteAgentActionRequest_DOWN: // DOWN
		if entity.Y == minPosition {
			// Update the entity
			err = s.datacomDAL.UpdateEntity(origionalContent, *entity)
			if err != nil {
				log.Printf("ERROR: %v\n", err)
			}
			return &envApi.ExecuteAgentActionResponse{
				Value: envApi.ExecuteAgentActionResponse_ERR_INVALID_TARGET,
			}, nil
		}
		targetY--
	case envApi.ExecuteAgentActionRequest_LEFT: // LEFT
		if entity.X == minPosition {
			// Update the entity
			err = s.datacomDAL.UpdateEntity(origionalContent, *entity)
			if err != nil {
				log.Printf("ERROR: %v\n", err)
			}
			return &envApi.ExecuteAgentActionResponse{
				Value: envApi.ExecuteAgentActionResponse_ERR_INVALID_TARGET,
			}, nil
		}
		targetX--
	case envApi.ExecuteAgentActionRequest_RIGHT: // RIGHT
		targetX++
	}

	// Create the response object
	var resp *envApi.ExecuteAgentActionResponse
	var respErr error

	switch req.Action {
	case 0: // REST
		// Set response to ok
		resp = &envApi.ExecuteAgentActionResponse{
			Value: envApi.ExecuteAgentActionResponse_OK,
		}
		break
	case 1: // MOVE
		if targetX > maxPosition || targetY > maxPosition {
			resp = &envApi.ExecuteAgentActionResponse{
				Value: envApi.ExecuteAgentActionResponse_ERR_INVALID_TARGET,
			}
			break
		}
		// Check if cell is occupied
		isCellOccupied, _, _, err := s.datacomDAL.IsCellOccupied(targetX, targetY)
		if isCellOccupied || err != nil {
			// Return unsuccessful
			resp = &envApi.ExecuteAgentActionResponse{
				Value: envApi.ExecuteAgentActionResponse_ERR_INVALID_TARGET,
			}
			break
		}
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
				s.datacomDAL.RemoveEntityMetadataFromFirebase(entity.Id)
				resp = &envApi.ExecuteAgentActionResponse{
					Value: envApi.ExecuteAgentActionResponse_ERR_DIED,
				}
				return resp, nil
			}
		}

		// Calculate scent
		// TODO: This is temporary and should probably be replaced with a generated number
		// on model creation
		scentString := ""
		added := 0 // limit on how many numbers to add
		for i := 0; i < len(entity.ModelID); i++ {
			if added == 5 {
				break
			}
			i, err := strconv.ParseInt("0x"+string(entity.ModelID[i]), 0, 32)
			if err == nil {
				scentString += strconv.Itoa(int(i))
				added++
			}
		}
		scentNum, _ := strconv.ParseInt(scentString, 0, 32)

		// Create pheromone effect
		s.datacomDAL.CreateEffect(envApi.Effect{
			X:         entity.X,
			Y:         entity.Y,
			ClassID:   envApi.Effect_Class(1),
			Value:     uint32(scentNum),
			Decay:     1.2,
			DelThresh: 5,
			Timestamp: time.Now().Unix(),
		})
		// Finally, adjust the position
		entity.X = targetX
		entity.Y = targetY
		// Set response to ok
		resp = &envApi.ExecuteAgentActionResponse{
			Value: envApi.ExecuteAgentActionResponse_OK,
		}
		break
	case 2: // EAT
		// Check if cell is occupied
		isCellOccupied, other, _, err := s.datacomDAL.IsCellOccupied(targetX, targetY)
		if !isCellOccupied || err != nil {
			// Return unsuccessful
			resp = &envApi.ExecuteAgentActionResponse{
				Value: envApi.ExecuteAgentActionResponse_ERR_INVALID_TARGET,
			}
			break
		}
		if other.ClassID != envApi.Entity_FOOD { // FOOD
			resp = &envApi.ExecuteAgentActionResponse{
				Value: envApi.ExecuteAgentActionResponse_ERR_INVALID_TARGET,
			}
			break
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
				respErr = err
				break
			}
			entityID := newUUID.String()
			// Create entity
			x, y := randomPosition()
			e := envApi.Entity{
				Id:      entityID,
				ClassID: 3,
				X:       x,
				Y:       y,
			}
			// Create entity silently (no publish)
			err = s.datacomDAL.CreateEntity(e, true)
			if err == nil {
				break
			}
		}
		resp = &envApi.ExecuteAgentActionResponse{
			Value: envApi.ExecuteAgentActionResponse_OK,
		}
		break
	case 3: // ATTACK
		// Check if cell is occupied
		isCellOccupied, other, otherOrigionalContent, err := s.datacomDAL.IsCellOccupied(targetX, targetY)
		if !isCellOccupied || err != nil {
			// Return unsuccessful
			resp = &envApi.ExecuteAgentActionResponse{
				Value: envApi.ExecuteAgentActionResponse_ERR_INVALID_TARGET,
			}
			break
		}
		// Make sure the other entity is an agent
		if other.ClassID != 1 { // AGENT
			resp = &envApi.ExecuteAgentActionResponse{
				Value: envApi.ExecuteAgentActionResponse_ERR_INVALID_TARGET,
			}
			break
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
				s.datacomDAL.RemoveEntityMetadataFromFirebase(entity.Id)
				resp = &envApi.ExecuteAgentActionResponse{
					Value: envApi.ExecuteAgentActionResponse_ERR_DIED,
				}
				return resp, nil
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
			s.datacomDAL.RemoveEntityMetadataFromFirebase(other.Id)
		}
		resp = &envApi.ExecuteAgentActionResponse{
			Value: envApi.ExecuteAgentActionResponse_OK,
		}
		break
	default: // INVALID
		resp = &envApi.ExecuteAgentActionResponse{
			Value: envApi.ExecuteAgentActionResponse_ERR_INVALID_TARGET,
		}
		break
	}

	if respErr != nil {
		return nil, respErr
	}

	// Update the entity
	err = s.datacomDAL.UpdateEntity(origionalContent, *entity)
	if err != nil {
		log.Printf("ERROR: %v\n", err)
	}

	// Return the data for the agent
	return resp, nil
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

	x0 := req.X * regionSize
	y0 := req.Y * regionSize
	x1 := x0 + regionSize - 1
	y1 := y0 + regionSize - 1

	entities, err := s.datacomDAL.GetEntitiesInSpace(x0, y0, x1, y1)
	if err != nil {
		log.Printf("ERROR: %v", err)
		return nil, err
	}

	return &envApi.GetEntitiesInRegionResponse{
		Entities: entities,
	}, nil
}

func (s *environmentServer) GetEffectsInRegion(ctx context.Context, req *envApi.GetEffectsInRegionRequest) (*envApi.GetEffectsInRegionResponse, error) {
	// Lock the data, defer unlock until end of call
	s.m.Lock()
	defer s.m.Unlock()
	effects := []*envApi.Effect{}

	x0 := req.X * regionSize
	y0 := req.Y * regionSize
	x1 := x0 + regionSize - 1
	y1 := y0 + regionSize - 1

	effects, err := s.datacomDAL.GetEffectsInSpace(x0, y0, x1, y1)
	if err != nil {
		log.Printf("ERROR: %v", err)
		return nil, err
	}

	return &envApi.GetEffectsInRegionResponse{
		Effects: effects,
	}, nil
}
