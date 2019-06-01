package collective

import (
	"fmt"
	"os"
	"sync"

	firebase "firebase.google.com/go"
	"github.com/go-redis/redis"
	api "github.com/terrariumai/simulation/pkg/api/collective"
	fb "github.com/terrariumai/simulation/pkg/fb"
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
	// TODO
	return nil
}
