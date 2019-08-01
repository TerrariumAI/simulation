package datacom

import (
	"encoding/json"
	"fmt"
	"log"

	pubnub "github.com/pubnub/go"
	envApi "github.com/terrariumai/simulation/pkg/api/environment"
)

// PubnubPAL specific struct for pubnub
type PubnubPAL struct {
	pubnubClient *pubnub.PubNub
	env          string
}

// NewPubnubPAL Creates a new pubnub specific Pubsub Access Layer
func NewPubnubPAL(env string, subkey string, pubkey string) PubsubAccessLayer {
	// Setup pubnub
	config := pubnub.NewConfig()
	config.SubscribeKey = subkey
	config.PublishKey = pubkey
	return &PubnubPAL{
		pubnubClient: pubnub.NewPubNub(config),
		env:          env,
	}
}

// PublishEvent publishes an event to pubnub for web clients to listen to
func (p *PubnubPAL) PublishEvent(eventName string, entity envApi.Entity) error {
	if p.env == "training" {
		return nil
	}
	b, err := json.Marshal(entity)
	if err != nil {
		log.Printf("PublishEvent(): error: %v\n", err)
		return err
	}

	msg := map[string]interface{}{
		"eventName":  eventName,
		"entityData": string(b),
	}

	x, y := getRegionForPos(entity.X, entity.Y)
	channel := fmt.Sprintf("%v.%v", x, y)
	_, _, err = p.pubnubClient.Publish().
		Channel(channel).Message(msg).Execute()

	if err != nil {
		log.Printf("PublishEvent(): error publishing: %v\n", err)
		return err
	}

	return nil
}
