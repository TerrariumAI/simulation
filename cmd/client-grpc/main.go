package main

import (
	"context"
	"flag"
	"log"
	"time"

	v1 "github.com/olamai/simulation/pkg/api/v1"
	"google.golang.org/grpc"
)

const (
	// apiVersion is version of API is provided by server
	apiVersion = "v1"
)

func main() {
	// get configuration
	address := flag.String("server", "", "gRPC server in format host:port")
	flag.Parse()

	// Set up a connection to the server.
	conn, err := grpc.Dial(*address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	c := v1.NewSimulationServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Call Create
	req1 := v1.CreateAgentRequest{
		Api: apiVersion,
		Agent: &v1.Agent{
			X: 0,
			Y: 0,
		},
	}
	res1, err := c.CreateAgent(ctx, &req1)
	if err != nil {
		log.Fatalf("Create failed: %v", err)
	}
	log.Printf("Create result: <%+v>\n\n", res1)

	id := res1.Id

	println("Id: ", id)

	// Get agent
	req2 := v1.GetAgentRequest{
		Api: apiVersion,
		Id:  id,
	}
	res2, err := c.GetAgent(ctx, &req2)
	if err != nil {
		log.Fatalf("Read failed: %v", err)
	}
	log.Printf("Read result: <%+v>\n\n", res2)
	println("Agent x: ", res2.Agent.X)
	println("Agent y: ", res2.Agent.Y)

	// Update
	req3 := v1.ExecuteAgentActionRequest{
		Api: apiVersion,
		Id:  id,
		Action: &v1.Action{
			Id:        "MOVE",
			Direction: "UP",
		},
	}
	res3, err := c.ExecuteAgentAction(ctx, &req3)
	if err != nil {
		log.Fatalf("Update failed: %v", err)
	}
	log.Printf("Update result: <%+v>\n\n", res3)

	// Call ReadAll
	req4 := v1.GetAgentObservationRequest{
		Api: apiVersion,
		Id:  id,
	}
	res4, err := c.GetAgentObservation(ctx, &req4)
	if err != nil {
		log.Fatalf("ReadAll failed: %v", err)
	}
	log.Printf("ReadAll result: <%+v>\n\n", res4)

	// // Delete
	// req5 := v1.DeleteRequest{
	// 	Api: apiVersion,
	// 	Id:  id,
	// }
	// res5, err := c.Delete(ctx, &req5)
	// if err != nil {
	// 	log.Fatalf("Delete failed: %v", err)
	// }
	// log.Printf("Delete result: <%+v>\n\n", res5)
}
