package environment

import (
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"

	"google.golang.org/grpc/metadata"

	uuid "github.com/satori/go.uuid"

	datacom "github.com/terrariumai/simulation/pkg/datacom"

	"github.com/golang/protobuf/ptypes/empty"
	api "github.com/terrariumai/simulation/pkg/api/environment"
)

const (
	// apiVersion is version of API is provided by server
	apiVersion            = "v1"
	agentLivingEnergyCost = 2
	minFoodBeforeRespawn  = 200
	regionSize            = 16
	maxPositionPadding    = 3
	maxPosition           = 999
)

// toDoServiceServer is implementation of api.ToDoServiceServer proto interface
type environmentServer struct {
	// Environment the server is running in
	env string
	// Datacom
	datacom *datacom.Datacom
	// Mutex to ensure data safety
	m sync.Mutex
}

// UserInfo is the struct that will parse the auth response
type UserInfo struct {
	Issuer string `json:"issuer"`
	ID     string `json:"id"`
	Email  string `json:"email"`
}

// PosToRedisIndex interlocks an x and y value to use as an
// index in redis
func PosToRedisIndex(x int32, y int32) (string, error) {
	// negatives are not allowed
	if x < 0 || y < 0 || x > maxPosition || y > maxPosition {
		return "", errors.New("Invalid position")
	}
	xString := strconv.Itoa(int(x))
	yString := strconv.Itoa(int(y))
	interlocked := ""
	// make sure x and y are the correct length when converted to str
	if len(xString) > maxPositionPadding || len(yString) > maxPositionPadding {
		return "", errors.New("X or Y position are too large")
	}
	// add padding
	for len(xString) < maxPositionPadding {
		xString = "0" + xString
	}
	for len(yString) < maxPositionPadding {
		yString = "0" + yString
	}
	// interlock
	for i := 0; i < maxPositionPadding; i++ {
		interlocked = interlocked + xString[i:i+1] + yString[i:i+1]
	}

	return interlocked, nil
}

// SerializeEntity takes in all the values for an entity and serializes them
//  to an entity content
func SerializeEntity(index string, x int32, y int32, class int32, ownerUID string, modelID string, id string) string {
	return fmt.Sprintf("%s:%v:%v:%v:%s:%s:%s", index, x, y, class, ownerUID, modelID, id)
}

// NewEnvironmentServer creates simulation service
func NewEnvironmentServer(env string, redisAddr string) api.EnvironmentServer {
	// initialize server
	s := &environmentServer{
		env: env,
	}

	datacom, err := datacom.NewDatacom(env, redisAddr)
	if err != nil {
		log.Fatalf("Error initializing Datacom: %v", err)
		os.Exit(1)
	}

	s.datacom = datacom

	return s
}

// Get data for an entity
func (s *environmentServer) CreateEntity(ctx context.Context, req *api.CreateEntityRequest) (*api.CreateEntityResponse, error) {
	// Lock the data, defer unlock until end of call
	s.m.Lock()
	defer s.m.Unlock()

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		fmt.Println("Not ok getting headers")
		return nil, nil
	}
	userInfoHeader := md["x-endpoint-api-userinfo"]
	sDec, _ := b64.StdEncoding.DecodeString(userInfoHeader[0])
	userInfo := UserInfo{}
	json.Unmarshal(sDec, &userInfo)

	// Make sure the user has supplied data
	if req.Entity == nil {
		return nil, errors.New("Entity not in request")
	}
	// Make sure the cell is not occupied
	isCellOccupied, err := s.datacom.IsCellOccupied(req.Entity.X, req.Entity.Y)
	if err != nil {
		return nil, err
	}
	if isCellOccupied {
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

	// Add the entity to the environment
	err = s.datacom.CreateEntity(req.Entity.X, req.Entity.Y, req.Entity.Class, userInfo.ID, req.Entity.ModelID, entityID)

	// Return the data for the agent
	return &api.CreateEntityResponse{
		Id: entityID,
	}, nil
}

// Get data for an entity
func (s *environmentServer) GetEntity(ctx context.Context, req *api.GetEntityRequest) (*api.GetEntityResponse, error) {
	// Lock the data, defer unlock until end of call
	s.m.Lock()
	defer s.m.Unlock()

	// Get the entity
	entity, _, err := s.datacom.GetEntity(req.Id)
	if err != nil {
		return nil, errors.New("Couldn't find an entity by that id")
	}

	// Return the data for the agent
	return &api.GetEntityResponse{
		Entity: entity,
	}, nil
}

// Get data for an entity
func (s *environmentServer) DeleteEntity(ctx context.Context, req *api.DeleteEntityRequest) (*api.DeleteEntityResponse, error) {
	// Lock the data, defer unlock until end of call
	s.m.Lock()
	defer s.m.Unlock()

	// Remove the entity from the environment
	deleted, err := s.datacom.DeleteEntity(req.Id)
	if err != nil {
		return nil, err
	}

	// Return the data for the agent
	return &api.DeleteEntityResponse{
		Deleted: deleted,
	}, nil
}

// Get data for an entity
func (s *environmentServer) ExecuteAgentAction(ctx context.Context, req *api.ExecuteAgentActionRequest) (*api.ExecuteAgentActionResponse, error) {
	// Lock the data, defer unlock until end of call
	s.m.Lock()
	defer s.m.Unlock()

	fmt.Printf("Execute agent action: %v \n", req)
	// Get the entity
	entity, origionalContent, err := s.datacom.GetEntity(req.Id)
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

	switch req.Action {
	case 0: // MOVE
		// Check if cell is occupied
		isCellOccupied, err := s.datacom.IsCellOccupied(targetX, targetY)
		if isCellOccupied || err != nil {
			// Return unsuccessful
			return &api.ExecuteAgentActionResponse{
				WasSuccessful: false,
			}, nil
		}
		// Update the entity
		err = s.datacom.UpdateEntity(*origionalContent, targetX, targetY, entity.Class, entity.OwnerUID, entity.ModelID, entity.Id)
		if err != nil {
			return nil, err
		}
	default: // INVALID
		return &api.ExecuteAgentActionResponse{
			WasSuccessful: false,
		}, nil
	}

	// Return the data for the agent
	return &api.ExecuteAgentActionResponse{
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

/*
getRegionForPos(p) {
    let x = p.x;
    let y = p.y;
    if (x < 0) {
      x -= CELLS_IN_REGION;
    }
    if (y < 0) {
      y -= CELLS_IN_REGION;
    }
    return {
      x:
        x <= 0
          ? Math.ceil(x / CELLS_IN_REGION)
          : Math.floor(x / CELLS_IN_REGION),
      y:
        y <= 0
          ? Math.ceil(y / CELLS_IN_REGION)
          : Math.floor(y / CELLS_IN_REGION)
    };
  }
*/
func (s *environmentServer) GetEntitiesInRegion(ctx context.Context, req *api.GetEntitiesInRegionRequest) (*api.GetEntitiesInRegionResponse, error) {
	// Lock the data, defer unlock until end of call
	s.m.Lock()
	defer s.m.Unlock()
	entities := []*api.Entity{}

	entities, err := s.datacom.GetEntitiesInRegion(req.X, req.Y)
	if err != nil {
		log.Printf("GetEntitiesInRegion(): error %v", err)
	}

	return &api.GetEntitiesInRegionResponse{
		Entities: entities,
	}, nil
}
