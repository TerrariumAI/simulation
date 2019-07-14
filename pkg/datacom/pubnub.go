package datacom

import (
	"encoding/json"
	"log"

	envApi "github.com/terrariumai/simulation/pkg/api/environment"
)

// PublishEvent publishes an event to pubnub for web clients to listen to
func (dc *Datacom) PublishEvent(eventName string, entity envApi.Entity) {
	b, err := json.Marshal(entity)
	if err != nil {
		log.Printf("PublishEvent(): error: %v\n", err)
		return
	}

	msg := map[string]interface{}{
		"eventName": eventName,
		"entity":    string(b),
	}

	_, _, err = dc.pubnubClient.Publish().
		Channel("hello_world").Message(msg).Execute()

	if err != nil {
		log.Printf("PublishEvent(): error publishing: %v\n", err)
	}
}
