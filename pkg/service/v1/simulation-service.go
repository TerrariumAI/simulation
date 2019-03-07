package v1

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	v1 "github.com/olamai/simulation/pkg/api/v1"
)

const (
	// apiVersion is version of API is provided by server
	apiVersion               = "v1"
	AGENT_LIVING_ENERGY_COST = 2
)

// toDoServiceServer is implementation of v1.ToDoServiceServer proto interface
type simulationServiceServer struct {
	// Entity storage
	entities map[string]*Entity
	// Map from position -> *Entity
	posEntityMap map[Vec2]*Entity
	// Map from spectator id -> observation channel
	spectIdChanMap map[string]chan v1.CellUpdate
	// Specators subscription to regions
	spectRegionSubs map[Vec2][]string
}

// NewSimulationServiceServer creates ToDo service
func NewSimulationServiceServer() v1.SimulationServiceServer {
	s := &simulationServiceServer{
		entities:        make(map[string]*Entity),
		posEntityMap:    make(map[Vec2]*Entity),
		spectIdChanMap:  make(map[string]chan v1.CellUpdate),
		spectRegionSubs: make(map[Vec2][]string),
	}

	// Spawn food randomly
	for i := 0; i < 100; i++ {
		x := int32(rand.Intn(50) - 25)
		y := int32(rand.Intn(50) - 25)
		// Don't put anything at 0,0
		if x == 0 || y == 0 {
			continue
		}
		s.NewEntity("FOOD", Vec2{x, y})
	}

	return s
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

// Broadcast a cell update
func (s *simulationServiceServer) BroadcastCellUpdate(pos Vec2, occupant string) {
	// TODO
}

// Create new agent
func (s *simulationServiceServer) CreateAgent(ctx context.Context, req *v1.CreateAgentRequest) (*v1.CreateAgentResponse, error) {
	// check if the API version requested by client is supported by server
	if err := s.checkAPI(req.Api); err != nil {
		return nil, err
	}

	targetPos := Vec2{req.Agent.X, req.Agent.Y}

	// Make sure the cell is empty
	if _, ok := s.posEntityMap[targetPos]; ok {
		err := errors.New("CreateAgent(): Cell is already occupied")
		return nil, err
	}

	// Create a new agent (which is an entity)
	agent := s.NewEntity("AGENT", Vec2{req.Agent.X, req.Agent.Y})

	// Broadcast update
	s.BroadcastCellUpdate(agent.pos, agent.class)

	return &v1.CreateAgentResponse{
		Api: apiVersion,
		Id:  agent.id,
	}, nil
}

// Get data for an agent
func (s *simulationServiceServer) GetAgent(ctx context.Context, req *v1.GetAgentRequest) (*v1.GetAgentResponse, error) {
	// check if the API version requested by client is supported by server
	if err := s.checkAPI(req.Api); err != nil {
		return nil, err
	}
	// Get the agent from the map
	agent, ok := s.entities[req.Id]
	// Throw an error if an agent by that id doesn't exist
	if !ok {
		err := errors.New("GetAgent(): Agent Not Found")
		return nil, err
	}

	// Return the data for the agent
	return &v1.GetAgentResponse{
		Api: apiVersion,
		Agent: &v1.Agent{
			Id: agent.id,
			X:  agent.pos.x,
			Y:  agent.pos.y,
		},
	}, nil
}

// Remove an agent
func (s *simulationServiceServer) DeleteAgent(ctx context.Context, req *v1.DeleteAgentRequest) (*v1.DeleteAgentResponse, error) {
	// check if the API version requested by client is supported by server
	if err := s.checkAPI(req.Api); err != nil {
		return nil, err
	}
	// Get the agent
	agent, ok := s.entities[req.Id]
	// Throw an error if an agent by that id doesn't exist
	if !ok {
		err := errors.New("GetAgent(): Agent Not Found")
		return nil, err
	}

	// Remove the entity
	s.RemoveEntityById(agent.id)

	// Return the data for the agent
	return &v1.DeleteAgentResponse{
		Api:     apiVersion,
		Deleted: req.Id,
	}, nil
}

// Execute an action for an agent
func (s *simulationServiceServer) ExecuteAgentAction(ctx context.Context, req *v1.ExecuteAgentActionRequest) (*v1.ExecuteAgentActionResponse, error) {
	action := req.Action
	var actionSuccess bool
	// check if the API version requested by client is supported by server
	if err := s.checkAPI(req.Api); err != nil {
		return nil, err
	}
	// Get the agent
	agent, ok := s.entities[req.Id]
	// Throw an error if an agent by that id doesn't exist
	if !ok {
		err := errors.New("GetAgent(): Agent Not Found")
		return nil, err
	}
	// Lower health immediatly if energy is 0
	if agent.energy == 0 {
		agent.health -= 10
	}
	// Kill the agent if they have no health and end call
	if agent.health <= 0 {
		s.RemoveEntityById(agent.id)
		return &v1.ExecuteAgentActionResponse{
			Api:                 apiVersion,
			IsAgentStillAlive:   false,
			WasActionSuccessful: false,
		}, nil
	}
	// Get the target position from the given direction
	var targetPos Vec2
	switch action.Direction {
	case "UP":
		targetPos = Vec2{agent.pos.x, agent.pos.y + 1}
	case "DOWN":
		targetPos = Vec2{agent.pos.x, agent.pos.y - 1}
	case "LEFT":
		targetPos = Vec2{agent.pos.x - 1, agent.pos.y}
	case "RIGHT":
		targetPos = Vec2{agent.pos.x + 1, agent.pos.y}
	default: // Direction not correct
		return nil, errors.New("ExecuteAgentAction(): Invalid Action.Direction")
	}

	// Perform the action
	switch action.Id {
	case "MOVE":
		actionSuccess = s.EntityMove(agent.id, targetPos)
	case "CONSUME":
		actionSuccess = s.EntityConsume(agent.id, targetPos)
	}

	// Take off living expense
	agent.energy -= AGENT_LIVING_ENERGY_COST
	if agent.energy < 0 {
		agent.energy = 0
	}

	return &v1.ExecuteAgentActionResponse{
		Api:                 apiVersion,
		IsAgentStillAlive:   true,
		WasActionSuccessful: actionSuccess,
	}, nil
}

// Get an observation for an agent
func (s *simulationServiceServer) GetAgentObservation(ctx context.Context, req *v1.GetAgentObservationRequest) (*v1.GetAgentObservationResponse, error) {
	// Get the agent
	e, ok := s.entities[req.Id]

	if ok {
		cells := s.GetObservationCellsForPosition(e.pos)
		// Agent is alive and well... maybe, at least it's alive
		return &v1.GetAgentObservationResponse{
			Api: apiVersion,
			Observation: &v1.Observation{
				Alive:  true,
				Cells:  cells,
				Energy: e.energy,
				Health: e.health,
			},
		}, nil
	} else {
		// Agent doesn't exist anymore
		return &v1.GetAgentObservationResponse{
			Api: apiVersion,
			Observation: &v1.Observation{
				Alive:  false,
				Cells:  []string{},
				Energy: 0,
				Health: 0,
			},
		}, nil
	}
}

// Remove an agent
func (s *simulationServiceServer) CreateSpectator(req *v1.CreateSpectatorRequest, stream v1.SimulationService_CreateSpectatorServer) error {
	// // Get info about the client
	// client, ok := peer.FromContext(stream.Context())
	// if !ok {
	// 	return errors.New("ERROR: Couldn't get info about peer")
	// }
	// addr := client.Addr.String()

	// Create a spectator id
	// For now it is just the ip of the client
	spectatorId := req.Id
	s.AddSpectatorChannel(spectatorId)

	// Listen for updates and send them to the client
	for {
		cellUpdate := <-s.spectIdChanMap[spectatorId]
		if err := stream.Send(&cellUpdate); err != nil {
			// Break the sending loop
			break
		}
	}

	// Remove the spectator and clean up
	log.Printf("Spectator left...")
	s.RemoveSpectatorChannel(spectatorId)

	return nil
}

// Get an observation for an agent
func (s *simulationServiceServer) SubscribeSpectatorToRegion(ctx context.Context, req *v1.SubscribeSpectatorToRegionRequest) (*v1.SubscribeSpectatorToRegionResponse, error) {
	// Get Headers
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.DataLoss, "SubscribeSpectatorToRegion(): UnaryEcho: failed to get metadata")
	}
	if token, ok := md["auth-token"]; ok {

		fmt.Printf("Custom header from metadata: " + token[0])
	}

	// customHeader := ctx.Value("custom-header=1")
	id := req.Id
	region := Vec2{req.Region.X, req.Region.Y}
	// If the user is already subbed, successful is false
	if s.isSpectatorAlreadySubscribedToRegion(id, region) {
		return &v1.SubscribeSpectatorToRegionResponse{
			Api:        apiVersion,
			Successful: false,
		}, nil
	}
	// Add spectator id to subscription slice
	s.spectRegionSubs[region] = append(s.spectRegionSubs[region], id)
	// Get spectator channel
	channel := s.spectIdChanMap[id]
	// Send initial world state
	xs, ys := region.GetPositionsInRegion()
	for _, x := range xs {
		for _, y := range ys {
			pos := Vec2{x, y}
			if entity, ok := s.posEntityMap[pos]; ok {
				channel <- v1.CellUpdate{X: pos.x, Y: pos.y, Occupant: entity.class}
			}
		}
	}

	return &v1.SubscribeSpectatorToRegionResponse{
		Api:        apiVersion,
		Successful: true,
	}, nil
}
