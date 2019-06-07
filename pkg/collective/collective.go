package collective

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	firebase "firebase.google.com/go"
	"github.com/go-redis/redis"
	uuid "github.com/satori/go.uuid"
	api "github.com/terrariumai/simulation/pkg/api/collective"
	envApi "github.com/terrariumai/simulation/pkg/api/environment"
	environment "github.com/terrariumai/simulation/pkg/environment"
	fb "github.com/terrariumai/simulation/pkg/fb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	mockModelID              = "MOCK-MODEL-ID"
	minFrameTimeMilliseconds = 50
)

// toDoServiceServer is implementation of api.ToDoServiceServer proto interface
type collectiveServer struct {
	// Environment the server is running in
	env string
	// --- Firebase ---
	// Firebase app
	firebaseApp *firebase.App
	// Mutex to ensure data safety
	m sync.Mutex
	// Redis client
	redisClient *redis.Client
	// Environment client
	envClient envApi.EnvironmentClient
}

// func parseEntityContent(content string) api.Entity {
// 	values := strings.Split(content, ":")
// 	return api.Entity{
// 		Class: values[3],
// 		Id:    values[6],
// 	}
// }

// NewCollectiveServer creates a new collective server
func NewCollectiveServer(env string, redisAddr string, envAddress string) api.CollectiveServer {
	// initialize redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	pong, err := redisClient.Ping().Result()
	if env != "prod" && err != nil {
		fmt.Println("Couldn't connect to redis: "+pong, err)
		os.Exit(1)
	}
	// initialize server
	s := &collectiveServer{
		env:         env,
		firebaseApp: fb.InitializeFirebaseApp(env),
		redisClient: redisClient,
	}
	// initialize env client
	conn, err := grpc.Dial(envAddress, grpc.WithInsecure())
	if err != nil {
		fmt.Println("Couldn't connect to environment service: "+pong, err)
		os.Exit(1)
	}
	envClient := envApi.NewEnvironmentClient(conn)
	s.envClient = envClient

	return s
}

func (s *collectiveServer) ConnectRemoteModel(stream api.Collective_ConnectRemoteModelServer) error {
	ctx := stream.Context()
	md, ok := metadata.FromIncomingContext(ctx)

	// Make sure name is in metadata
	if !ok {
		return errors.New("ConnectRemoteModel(): Error parsing metadata")
	}
	nameHeader, ok := md["model-name"]
	if !ok {
		return errors.New("ConnectRemoteModel(): No name in context")
	}
	name := nameHeader[0]

	// Add the RM to the database
	modelID := uuid.NewV4().String()
	if s.env == "testing" {
		modelID = mockModelID
	}
	err := s.redisClient.HSet("model:"+modelID+":metadata", "name", name).Err()
	if err != nil {
		return err
	}

	// Once the model is disconnected, remove it's data from the server
	defer s.cleanupModel(modelID)

	// Start the loop
	for {
		// Get the entitiy IDs for this model
		entityIdsRequest := s.redisClient.SMembers("model:" + modelID + ":entities")
		if err := entityIdsRequest.Err(); err != nil {
			log.Fatalf("ConnectRemoteModel(): %v", err)
			return errors.New("ConnectRemoteModel(): Couldn't access the database")
		}
		entityIds := entityIdsRequest.Val()
		// Get the entities for this model
		entitiesContentRequest := s.redisClient.HMGet("entities.content", entityIds...)
		if entitiesContentRequest.Err() != nil {
			return errors.New("ConnectRemoteModel(): Couldn't access the database")
		}
		entitiesContent := entitiesContentRequest.Val()
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
			indexMin, err := environment.PosToRedisIndex(xMin, yMin)
			if err != nil {
				return fmt.Errorf("Error converting min/max positions to index: %v", err)
			}
			indexMax, err := environment.PosToRedisIndex(xMax, yMax)
			if err != nil {
				return fmt.Errorf("Error converting min/max positions to index: %v", err)
			}
			println("Performing range query from: ", indexMin, " to ", indexMax)
			// Perform the query
			rangeQuery := s.redisClient.ZRangeByLex("entities", redis.ZRangeBy{
				Min: "(" + indexMin,
				Max: "(" + indexMax,
			})
			if err := rangeQuery.Err(); err != nil {
				return fmt.Errorf("Error in range query: %v", err)
			}
			closeEntitiesContent := rangeQuery.Val()
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

		// Send the observation packet
		if err := stream.Send(&obsvPacket); err != nil {
			// TODO - Clean disconnect, remove data from database
			return err
		}

		// Wait for a response
		actionPacket, err := stream.Recv()
		if err == io.EOF {
			return err
		}

		println("Received Something from the client!")

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
			fmt.Printf("Sending action request: %v \n", req)
			_, err := s.envClient.ExecuteAgentAction(ctx, &req)
			if err != nil {
				fmt.Printf("Error sending action: %v \n: ", err)
				return err
			}
		}

		// Wait if we got a response too quickly
		t2 := time.Now().UnixNano() / 1000000
		delta := t2 - t1
		if delta < minFrameTimeMilliseconds {
			time.Sleep(time.Duration((minFrameTimeMilliseconds - delta)) * time.Millisecond)
		}
	}
}

func (s *collectiveServer) cleanupModel(modelID string) {
	println("Cleaning up model... model:", modelID)
	err := s.redisClient.Del("model:" + modelID + ":entities").Err()
	if err != nil {
		fmt.Printf("Error cleaning up model entities: %v \n", err)
	}
	err = s.redisClient.Del("model:" + modelID + ":metadata").Err()
	if err != nil {
		fmt.Printf("Error cleaning up model metadata: %v \n", err)
	}
}
