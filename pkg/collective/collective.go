package collective

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	firebase "firebase.google.com/go"
	"github.com/go-redis/redis"
	uuid "github.com/satori/go.uuid"
	api "github.com/terrariumai/simulation/pkg/api/collective"
	environment "github.com/terrariumai/simulation/pkg/environment"
	fb "github.com/terrariumai/simulation/pkg/fb"
	"google.golang.org/grpc/metadata"
)

const (
	mockModelID = "MOCK-MODEL-ID"
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
}

func parseEntityContent(content string) api.Entity {
	values := strings.Split(content, ":")
	return api.Entity{
		Class: values[3],
		Id:    values[6],
	}
}

// NewCollectiveServer creates a new collective server
func NewCollectiveServer(env string) api.CollectiveServer {
	// initialize redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
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

	return s
}

func (s *collectiveServer) ConnectRemoteModel(stream api.Collective_ConnectRemoteModelServer) error {
	md, ok := metadata.FromIncomingContext(stream.Context())

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
		for _, content := range entitiesContent {
			// entity := parseEntityContent(content.(string))
			entity := environment.ParseEntityContent(content.(string))
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
			closeEntities := rangeQuery.Val()
			fmt.Println(closeEntities)
		}
		println(entitiesContent)
		return nil
		// in, err := stream.Recv()
		// if err == io.EOF {
		// 	return nil
		// }
		// if err != nil {
		// 	return err
		// }
		// key := serialize(in.Location)
		//             ... // look for notes to be sent to client
		// for _, note := range s.routeNotes[key] {
		// 	if err := stream.Send(note); err != nil {
		// 		return err
		// 	}
		// }
	}
	return nil
}
