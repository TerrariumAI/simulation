package datacom

import (
	"encoding/json"
	"fmt"
	"log"
	"math"

	envApi "github.com/terrariumai/simulation/pkg/api/environment"
)

const cellsInRegion float64 = 10

func getRegionForPos(x int32, y int32) (int32, int32) {
	regionX := x
	regionY := y
	if x < 0 {
		x -= int32(cellsInRegion)
	}
	if y < 0 {
		y -= int32(cellsInRegion)
	}

	if x <= 0 {
		regionX = int32(math.Ceil(float64(x) / cellsInRegion))
	} else {
		regionX = int32(math.Floor(float64(x) / cellsInRegion))
	}

	if y <= 0 {
		regionY = int32(math.Ceil(float64(y) / cellsInRegion))
	} else {
		regionY = int32(math.Floor(float64(y) / cellsInRegion))
	}

	return regionX, regionY
}

// PublishEvent publishes an event to pubnub for web clients to listen to
func (dc *Datacom) PublishEvent(eventName string, entity envApi.Entity) {
	b, err := json.Marshal(entity)
	if err != nil {
		log.Printf("PublishEvent(): error: %v\n", err)
		return
	}

	msg := map[string]interface{}{
		"eventName":  eventName,
		"entityData": string(b),
	}

	x, y := getRegionForPos(entity.X, entity.Y)
	channel := fmt.Sprintf("%v.%v", x, y)
	_, _, err = dc.pubnubClient.Publish().
		Channel(channel).Message(msg).Execute()

	if err != nil {
		log.Printf("PublishEvent(): error publishing: %v\n", err)
	}
}
