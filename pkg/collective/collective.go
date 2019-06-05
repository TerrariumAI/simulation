package collective

import (
	"errors"
	"fmt"
	"os"
	"sync"

	firebase "firebase.google.com/go"
	"github.com/go-redis/redis"
	uuid "github.com/satori/go.uuid"
	api "github.com/terrariumai/simulation/pkg/api/collective"
	fb "github.com/terrariumai/simulation/pkg/fb"
	"google.golang.org/grpc/metadata"
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
	println("Started")
	md, ok := metadata.FromIncomingContext(stream.Context())
	println("Got context")

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
	err := s.redisClient.HSet("model:"+modelID, "name", name).Err()

	if err != nil {
		return err
	}
	// TODO
	for {
		// Get the entities for this model
		entityIdsRequest := s.redisClient.SMembers("model:" + modelID)
		if err := entityIdsRequest; err != nil {
			return errors.New("ConnectRemoteModel(): Couldn't access the database")
		}
		entityIds := entityIdsRequest.Val()
		print(entityIds)
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
