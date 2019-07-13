package collective

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	datacom "github.com/terrariumai/simulation/pkg/datacom"

	api "github.com/terrariumai/simulation/pkg/api/collective"
	envApi "github.com/terrariumai/simulation/pkg/api/environment"
	environment "github.com/terrariumai/simulation/pkg/environment"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	minFrameTimeMilliseconds = 50
)

// toDoServiceServer is implementation of api.ToDoServiceServer proto interface
type collectiveServer struct {
	// Environment the server is running in
	env string
	// Mutex to ensure data safety
	m sync.Mutex
	// Datacom
	datacom *datacom.Datacom
	// Environment client
	envClient envApi.EnvironmentClient
}

// UserInfo is the struct that will parse the auth response
type UserInfo struct {
	Issuer string `json:"issuer"`
	ID     string `json:"id"`
	Email  string `json:"email"`
}

// NewCollectiveServer creates a new collective server
func NewCollectiveServer(env string, redisAddr string, envAddress string) api.CollectiveServer {
	// Init datacom
	datacom, err := datacom.NewDatacom(env, redisAddr)
	if err != nil {
		log.Fatalf("Error initializing Datacom: %v", err)
	}

	// Init environment client
	conn, err := grpc.Dial(envAddress, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Couldn't connect to environment service: %v", err)
	}
	envClient := envApi.NewEnvironmentClient(conn)

	// Init server
	s := &collectiveServer{
		env:       env,
		datacom:   datacom,
		envClient: envClient,
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
	remoteModelMD, err := s.datacom.GetRemoteModelMetadataForUser(modelSecret)
	if err != nil {
		log.Printf("Error: %v\n", err)
		return fmt.Errorf("ConnectRemoteModel(): That model does not exist or invalid secret key: %v", err)
	}

	defer s.cleanupModel(remoteModelMD.ID)

	log.Println("Starting loop")
	sendt1 := time.Now().UnixNano() / 1000000
	// Start the loop
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		// Query db for entities
		entitiesContent, err := s.datacom.GetEntitiesForModel(remoteModelMD.ID)
		if err != nil {
			return err
		}
		// Create a new observation packet to send
		var obsvPacket api.ObservationPacket

		// Generate an observation for each entity
		for _, content := range entitiesContent {
			entity, _ := environment.ParseEntityContent(content.(string))
			obsv := api.Observation{
				Id: entity.Id,
			}
			xMin := entity.X - 1
			xMax := entity.X + 1
			yMin := entity.Y - 1
			yMax := entity.Y + 1
			// Query for entities near this position
			closeEntitiesContent, err := s.datacom.GetEntitiesAroundPosition(xMin, yMin, xMax, yMax)
			if err != nil {
				return err
			}
			// Add all the other entities to the indexEntityMap
			// Match them up with the correct positions
			indexEntityMap := make(map[string]envApi.Entity)
			for _, otherContent := range closeEntitiesContent {
				// Don't count the same entity
				if content.(string) == otherContent {
					continue
				}
				otherEntity, index := environment.ParseEntityContent(content.(string))
				indexEntityMap[index] = otherEntity
			}
			for y := entity.Y - 1; y < entity.Y+1; y++ {
				for x := entity.X - 1; x < entity.X+1; x++ {
					index, err := environment.PosToRedisIndex(x, y)
					if err != nil {
						obsv.Cells = append(obsv.Cells, &api.Entity{Id: "", Class: 0})
						continue
					}
					if otherEntity, ok := indexEntityMap[index]; ok {
						obsv.Cells = append(obsv.Cells, &api.Entity{Id: otherEntity.Id, Class: otherEntity.Class})
					} else {
						obsv.Cells = append(obsv.Cells, &api.Entity{Id: "", Class: 0})
					}
				}
			}
			obsvPacket.Observations = append(obsvPacket.Observations, &obsv)
		}

		// We want to get the current time when we send the observation so
		//  we can check the difference when we get a response. If the resp
		//  comes sooner than the minimum frame time, we will wait
		t1 := time.Now().UnixNano() / 1000000

		// Only attempt any logic if there are observations to send
		if len(obsvPacket.Observations) > 0 {
			sendt2 := time.Now().UnixNano() / 1000000
			// Send the observation packet
			sendtDur1 := time.Now().UnixNano() / 1000000
			if err := stream.Send(&obsvPacket); err != nil {
				// TODO - Clean disconnect, remove data from database
				return err
			}
			sendtDur2 := time.Now().UnixNano() / 1000000
			sendDurDiff := sendtDur2 - sendtDur1
			println("SendDurDiff: ", sendDurDiff)

			diff := sendt2 - sendt1
			println("SendDiff: ", diff)
			sendt1 = sendt2

			// Wait for a response
			respDur1 := time.Now().UnixNano() / 1000000
			actionPacket, err := stream.Recv()
			if err == io.EOF {
				return err
			}
			respDur2 := time.Now().UnixNano() / 1000000
			respDurDiff := respDur2 - respDur1
			println("RespDurDiff: ", respDurDiff)

			// Perform actions
			actions := actionPacket.GetActions()
			md := metadata.Pairs("auth-secret", "MOCK-SECRET")
			ctx := metadata.NewOutgoingContext(context.Background(), md)
			for _, action := range actions {
				req := envApi.ExecuteAgentActionRequest{
					Id:        action.Id,
					Action:    action.Action,
					Direction: action.Direction,
				}
				_, err := s.envClient.ExecuteAgentAction(ctx, &req)
				if err != nil {
					fmt.Printf("Error sending action: %v \n: ", err)
					return err
				}
			}
		}

		// Wait if we got a response too quickly
		t2 := time.Now().UnixNano() / 1000000
		delta := t2 - t1
		if delta < minFrameTimeMilliseconds {
			// println("waiting for ", minFrameTimeMilliseconds-delta)
			time.Sleep(time.Duration((minFrameTimeMilliseconds - delta)) * time.Millisecond)
		}
	}
}

func (s *collectiveServer) cleanupModel(modelID string) {
	println("Cleaning up model... model:", modelID)
	// err := s.redisClient.Del("model:" + modelID + ":entities").Err()
	// if err != nil {
	// 	fmt.Printf("Error cleaning up model entities: %v \n", err)
	// }
}
