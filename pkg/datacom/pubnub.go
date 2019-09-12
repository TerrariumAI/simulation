package datacom

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/golang/protobuf/proto"
	pubnub "github.com/pubnub/go"
)

const (
	publishDelay = 250
)

type pubMsg struct {
	Channel string
	Msg     map[string]interface{}
}

type batchPubMsg struct {
	Events []interface{}
}

// PubnubPAL specific struct for pubnub
type PubnubPAL struct {
	pubnubClient *pubnub.PubNub
	env          string
	pubChan      chan pubMsg
}

// NewPubnubPAL Creates a new pubnub specific Pubsub Access Layer
func NewPubnubPAL(env string, subkey string, pubkey string) PubsubAccessLayer {
	// Setup pubnub
	config := pubnub.NewConfig()
	config.SubscribeKey = subkey
	config.PublishKey = pubkey
	p := PubnubPAL{
		pubnubClient: pubnub.NewPubNub(config),
		env:          env,
		pubChan:      make(chan pubMsg, 99),
	}

	// Start publish loop
	if env != "training" && env != "testing" {
		go p.StartBatchPublishLoop()
	}

	return &p
}

// QueuePublishEvent queues an event to be published as a batch later in Publisher
func (p *PubnubPAL) QueuePublishEvent(eventName string, protoMsg proto.Message, x uint32, y uint32) error {
	// Do nothing if we are training
	if p.env == "training" {
		return nil
	}

	// marshal
	b, err := json.Marshal(protoMsg)
	if err != nil {
		log.Printf("PublishEvent(): error: %v\n", err)
		return err
	}

	msg := map[string]interface{}{
		"eventName":  eventName,
		"entityData": string(b),
	}

	regionX, regionY := getRegionForPos(x, y)
	channel := fmt.Sprintf("%v-%v", regionX, regionY)

	p.pubChan <- pubMsg{
		channel,
		msg,
	}

	return nil
}

func (p *PubnubPAL) PublishMessage(channel string, message interface{}) error {
	_, _, err := p.pubnubClient.Publish().
		Channel(channel).Message(message).Execute()

	return err
}

func (p *PubnubPAL) BatchPublish() {
	if len(p.pubChan) == 0 {
		return
	}
	// maps regionId -> batchMessage
	batchMap := make(map[string]*batchPubMsg)

	// process all messages in channel
	for len(p.pubChan) > 0 {
		msg := <-p.pubChan
		if b, ok := batchMap[msg.Channel]; ok { // batch var already exists
			b.Events = append(b.Events, msg.Msg)
		} else { // batch var needs to be created
			// create the new batch holder
			b := batchPubMsg{
				Events: []interface{}{msg.Msg},
			}
			// add it to the map
			batchMap[msg.Channel] = &b
		}
	}

	// send batches
	for channel, batch := range batchMap {
		err := p.PublishMessage(channel, batch)

		if err != nil {
			log.Printf("ERROR: issue publishing batch: %v\n", err)
		}
	}
}

// StartBatchPublishLoop starts a loop that constantly publishes in batches per region,
//   then waits publishDelay milliseconds
func (p *PubnubPAL) StartBatchPublishLoop() {
	for {
		// publish
		p.BatchPublish()
		// sleep
		time.Sleep(publishDelay * time.Millisecond)
	}
}
