package datacom

import (
	"errors"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"

	envApi "github.com/terrariumai/simulation/pkg/api/environment"
)

// PosToRedisIndex interlocks an x and y value to use as an index in redis
func posToRedisIndex(x uint32, y uint32) (string, error) {
	xString := strconv.FormatUint(uint64(x), 2)
	yString := strconv.FormatUint(uint64(y), 2)
	interlocked := ""
	// make sure x and y are the correct length when converted to str
	if len(xString) > maxPositionCharLength || len(yString) > maxPositionCharLength {
		return "", errors.New("invalid position")
	}
	// add padding
	for len(xString) < maxPositionCharLength {
		xString = "0" + xString
	}
	for len(yString) < maxPositionCharLength {
		yString = "0" + yString
	}
	// interlock
	for i := 0; i < maxPositionCharLength; i++ {
		interlocked = interlocked + xString[i:i+1] + yString[i:i+1]
	}

	return interlocked, nil
}

// Takes in all the values for an entity and serializes them
//  to an entity content
// Returns error if the position is too large
func serializeEntity(e envApi.Entity) (string, error) {
	index, err := posToRedisIndex(e.X, e.Y)
	if err != nil {
		log.Println("ERROR: ", err)
		return "", err
	}
	return fmt.Sprintf("%s:%v:%v:%v:%s:%s:%v:%v:%s", index, e.X, e.Y, e.ClassID, e.OwnerUID, e.ModelID, e.Energy, e.Health, e.Id), nil
}

// ParseEntityContent takes entity content and parses it out to an entity
func parseEntityContent(content string) (entity envApi.Entity, index string) {
	values := strings.Split(content, ":")
	x, _ := strconv.Atoi(values[1])
	y, _ := strconv.Atoi(values[2])
	classID, _ := strconv.Atoi(values[3])
	ownerUID := values[4]
	modelID := values[5]
	energy, _ := strconv.Atoi(values[6])
	health, _ := strconv.Atoi(values[7])
	return envApi.Entity{
		X:        uint32(x),
		Y:        uint32(y),
		ClassID:  uint32(classID),
		OwnerUID: ownerUID,
		ModelID:  modelID,
		Energy:   uint32(energy),
		Health:   uint32(health),
		Id:       values[8],
	}, values[0]
}

func getRegionForPos(x uint32, y uint32) (uint32, uint32) {
	regionX := uint32(math.Floor(float64(x) / cellsInRegion))
	regionY := uint32(math.Floor(float64(y) / cellsInRegion))

	return regionX, regionY
}
