package collective

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	datacom "github.com/terrariumai/simulation/pkg/datacom"

	api "github.com/terrariumai/simulation/pkg/api/collective"
	envApi "github.com/terrariumai/simulation/pkg/api/environment"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	defaultMinStepTimeMilliseconds int64 = 250 // 50
	entitiesPollWaitTime           int64 = 250
	newGenerationPenaltyWaitTime   int64 = 1000
)

type collectiveServer struct {
	// Environment the server is running in
	env string
	// Mutex to ensure data safety
	m sync.Mutex
	// Datacom
	datacom *datacom.Datacom
	// Environment client
	envClient envApi.EnvironmentClient
	// Minimum each step should take
	minStepTimeMilliseconds int64
}

// UserInfo is the struct that will parse the auth response
type UserInfo struct {
	Issuer string `json:"issuer"`
	ID     string `json:"id"`
	Email  string `json:"email"`
}

// NewCollectiveServer creates a new collective server
func NewCollectiveServer(env string, redisAddr string, envAddress string, p datacom.PubsubAccessLayer) api.CollectiveServer {
	// Init datacom
	datacom, err := datacom.NewDatacom(env, redisAddr, p)
	if err != nil {
		log.Fatalf("Error initializing Datacom: %v", err)
	}

	// Init environment client
	conn, err := grpc.Dial(envAddress, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Couldn't connect to environment service: %v", err)
	}
	envClient := envApi.NewEnvironmentClient(conn)

	minStepTimeMilliseconds := defaultMinStepTimeMilliseconds
	// Disavle minimum step time in training, I WANNA GO FAST!
	if env == "training" {
		minStepTimeMilliseconds = 0
	}

	// Init server
	s := &collectiveServer{
		env:                     env,
		datacom:                 datacom,
		envClient:               envClient,
		minStepTimeMilliseconds: minStepTimeMilliseconds,
	}

	return s
}

func (s *collectiveServer) ConnectRemoteModel(stream api.Collective_ConnectRemoteModelServer) error {
	log.Println("Remote model has connected! Authorizing...")
	ctx := stream.Context()
	// Get metadata and parse userinfo
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		err := errors.New("ConnectRemoteModel(): Error getting metadata")
		log.Printf("Error: %v\n", err)
		return err
	}

	// Get secret header
	modelSecretHeader := md["model-secret"]
	if len(modelSecretHeader) == 0 {
		err := errors.New("ConnectRemoteModel(): authentication or model-secret header are missing")
		log.Printf("Error: %v\n", err)
		return err
	}
	modelSecret := modelSecretHeader[0]

	// Get RM metadata to make sure it exists
	remoteModelMD, err := s.datacom.GetRemoteModelMetadataBySecret(modelSecret)
	if err != nil {
		log.Printf("Error: %v\n", err)
		return fmt.Errorf("ConnectRemoteModel(): That model does not exist or invalid secret key: %v", err)
	}

	defer s.cleanupModel(remoteModelMD.ID)

	// Update the RM to show that this has connected
	s.datacom.UpdateRemoteModelMetadata(remoteModelMD, remoteModelMD.ConnectCount+1)

	// Dead entity observation array
	entityDeathObsvs := []api.Observation{}
	// Store memory of previous action
	entityActionResponseMemory := make(map[string]envApi.ExecuteAgentActionResponse)

	// Start the loop
	for {
		select {
		case <-ctx.Done():
			log.Println("WARNING: RM client disconnected")
			return nil
		default:
		}

		// Query db for entities
		entities, err := s.datacom.GetEntitiesForModel(remoteModelMD.ID)
		if err != nil {
			log.Printf("ERROR querying entities: %v\n", err)
			return err
		}

		// If there are no entities, wait the default poll time and then restart the loop
		if len(entities) == 0 {
			// Wait the penalty
			time.Sleep(time.Duration(newGenerationPenaltyWaitTime) * time.Millisecond)
			// Spawn entity with invalid x to force random placement
			s.envClient.CreateEntity(ctx, &envApi.CreateEntityRequest{Entity: &envApi.Entity{X: 999999999, ModelID: remoteModelMD.ID, ClassID: envApi.Entity_AGENT}})
		}

		// Create a new observation packet to send
		var obsvPacket api.ObservationPacket

		// Generate an observation for each entity
		for _, e := range entities {
			obsv, err := s.datacom.GetObservationForEntity(e)
			// Check for memory
			if resp, ok := entityActionResponseMemory[obsv.Id]; ok {
				obsv.ActionMemory = api.Observation_ResponseValue(resp.Value)
			}
			if err != nil {
				log.Printf("ERROR: %v\n", err)
			}
			obsvPacket.Observations = append(obsvPacket.Observations, obsv)
		}

		// Clear memory to ensure no memory leaks (naming, confusing, not sure where i am anymore, send help)
		entityActionResponseMemory = make(map[string]envApi.ExecuteAgentActionResponse)

		// Append death observations
		if len(entityDeathObsvs) > 0 {
			for _, obsv := range entityDeathObsvs {
				obsvPacket.Observations = append(obsvPacket.Observations, &obsv)
			}
			// Reset the slice
			entityDeathObsvs = nil
		}

		// We want to get the current time when we send the observation so
		//  we can check the difference when we get a response. If the resp
		//  comes sooner than the minimum frame time, we will wait
		t1 := time.Now().UnixNano() / 1000000

		// Only attempt any logic if there are observations to send
		if len(obsvPacket.Observations) > 0 {
			if err := stream.Send(&obsvPacket); err != nil {
				// TODO - Clean disconnect, remove data from database
				log.Printf("ERROR sending observation packet: %v\n", err)
				return err
			}

			// Wait for a response
			actionPacket, err := stream.Recv()
			if err != nil {
				fmt.Printf("ERROR: %v\n", err)
				return err
			}

			// Perform actions
			actions := actionPacket.GetActions()
			ctx := context.Background()
			for _, action := range actions {
				req := envApi.ExecuteAgentActionRequest{
					Id:        action.Id,
					Action:    envApi.ExecuteAgentActionRequest_Action(action.Action),
					Direction: envApi.ExecuteAgentActionRequest_Direction(action.Direction),
				}
				resp, err := s.envClient.ExecuteAgentAction(ctx, &req)
				if err != nil { // Note: Most often due to a message sent to a dead agent
					log.Printf("ERROR: %v\n", err)
					continue
				}
				// Store the response in memory
				entityActionResponseMemory[action.Id] = *resp
				// Check if the agent died during this action
				if resp.Value == envApi.ExecuteAgentActionResponse_ERR_DIED {
					// Add this observaion to the death obsvs slice to be used in the next loop
					entityDeathObsvs = append(entityDeathObsvs, api.Observation{
						Id:      action.Id,
						IsAlive: false,
					})
				}
			}
		}

		// Wait if we got a response too quickly
		t2 := time.Now().UnixNano() / 1000000
		delta := t2 - t1

		if delta < s.minStepTimeMilliseconds {
			sleepTime := time.Duration((s.minStepTimeMilliseconds - delta)) * time.Millisecond
			time.Sleep(sleepTime)
		}
	}
}

func (s *collectiveServer) cleanupModel(id string) {
	// Get RM metadata to make sure it exists
	remoteModelMD, err := s.datacom.GetRemoteModelMetadataByID(id)
	if err != nil {
		log.Printf("WARNING: Couldn't clean up model because it was not found id=%s\n", id)
		return
	}
	// Update the RM to show that this has disconnected
	err = s.datacom.UpdateRemoteModelMetadata(remoteModelMD, remoteModelMD.ConnectCount-1)
	if err != nil {
		log.Printf("WARNING: Couldn't clean up model, error updating metadata in firebase id=%s\n", id)
		return
	}
}
