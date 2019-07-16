package datacom

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	firebase "firebase.google.com/go"
	pubnub "github.com/pubnub/go"

	"github.com/go-redis/redis"
	envApi "github.com/terrariumai/simulation/pkg/api/environment"
	"google.golang.org/api/option"
)

const (
	serviceAccountProdFileLocation    = "./serviceAccountKey.json"
	serviceAccountStagingFileLocation = "./serviceAccountKey_staging.json"

	mockSecret = "MOCK-SECRET"

	maxPositionPadding = 3
	maxPosition        = 999

	regionSize = 10
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
	pubnubClient *pubnub.PubNub
}

// RemoteModel struct for parsing and storing RM data from databases
type RemoteModel struct {
	ID      string `firestore:"id,omitempty"`
	OwnerID string `firestore:"ownerId,omitempty"`
	Name    string `firestore:"name,omitempty"`
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

// ParseEntityContent takes entity content and parses it out to an entity
func ParseEntityContent(content string) (envApi.Entity, string) {
	values := strings.Split(content, ":")
	x, _ := strconv.Atoi(values[1])
	y, _ := strconv.Atoi(values[2])
	class, _ := strconv.Atoi(values[3])
	return envApi.Entity{
		X:        int32(x),
		Y:        int32(y),
		Class:    int32(class),
		OwnerUID: values[4],
		ModelID:  values[5],
		Id:       values[6],
	}, values[0]
}

// NewDatacom instantiates a new datacom object with proper clients
// according to the environment
func NewDatacom(env string, redisAddr string) (*Datacom, error) {
	dc := &Datacom{
		env: env,
	}

	// If we are training, we don't ever connect to any servers
	if env == "training" {
		return dc, nil
	}

	// Setup Firebase
	switch env {
	case "staging":
		// FIREBASE STAGING
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

	// Setup pubnub
	config := pubnub.NewConfig()
	config.SubscribeKey = "sub-c-b4ba4e28-a647-11e9-ad2c-6ad2737329fc"
	config.PublishKey = "pub-c-83ed11c2-81e1-4d7f-8e94-0abff2b85825"
	dc.pubnubClient = pubnub.NewPubNub(config)

	return dc, nil
}
