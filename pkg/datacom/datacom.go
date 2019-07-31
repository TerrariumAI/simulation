package datacom

import (
	"context"
	"log"
	"os"

	firebase "firebase.google.com/go"

	"github.com/go-redis/redis"
	envApi "github.com/terrariumai/simulation/pkg/api/environment"
	"google.golang.org/api/option"
)

const (
	serviceAccountProdFileLocation    = "./serviceAccountKey.json"
	serviceAccountStagingFileLocation = "./serviceAccountKey_staging.json"

	mockSecret = "MOCK-SECRET"

	minPosition        = 1
	maxPositionPadding = 3
	maxPosition        = 999

	regionSize            = 10
	cellsInRegion float64 = 10
)

// Datacom is an object that makes it easy to communicate with our
// databases. It handles figuring out where each specific
// bit of data is (redis, firebase, etc.) and how to access it (auth).
type Datacom struct {
	// current envirinment
	env string
	// firebase client
	firebaseApp *firebase.App
	// redis client
	redisClient *redis.Client
	// pubnub client
	pubsub PubsubAccessLayer
}

// RemoteModel struct for parsing and storing RM data from databases
type RemoteModel struct {
	ID           string `firestore:"id,omitempty"`
	OwnerUID     string `firestore:"ownerUID,omitempty"`
	Name         string `firestore:"name,omitempty"`
	ConnectCount int    `firestore:"connectCount,omitempty"`
}

// -------------
// Access Layers
// -------------
// Note: Access layers are interfaces that will hold generic actions for a specific service (pubsub, database, etc.)
//   This allows us to create a default implementation, AND mock easily.

// PubsubAccessLayer generic interface for pubsub services.
type PubsubAccessLayer interface {
	PublishEvent(eventName string, entity envApi.Entity) error
}

// NewDatacom instantiates a new datacom object with proper clients
// according to the environment
func NewDatacom(env string, redisAddr string, pubsub PubsubAccessLayer) (*Datacom, error) {
	dc := &Datacom{
		env:    env,
		pubsub: pubsub,
	}

	// Setup Firebase
	switch env {
	case "staging":
		// FIREBASE STAGING
		if _, err := os.Stat(serviceAccountStagingFileLocation); os.IsNotExist(err) {
			// path/to/whatever does not exist
			log.Panic("ERROR: Staging service account file not found")
		}
		opt := option.WithCredentialsFile(serviceAccountStagingFileLocation)
		app, err := firebase.NewApp(context.Background(), nil, opt)
		if err != nil {
			return nil, err
		}
		dc.firebaseApp = app
	case "prod":
		// FIREBASE PROD
		opt := option.WithCredentialsFile(serviceAccountProdFileLocation)
		app, err := firebase.NewApp(context.Background(), nil, opt)
		if err != nil {
			return nil, err
		}
		dc.firebaseApp = app
	}

	// Setup Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	_, err := redisClient.Ping().Result()
	if err != nil {
		return nil, err
	}
	dc.redisClient = redisClient

	return dc, nil
}
