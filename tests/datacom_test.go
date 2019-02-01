package tests

import (
	"log"

	"testing"

	pb "github.com/olamai/proto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func TestDatacom(t *testing.T) {
	var conn *grpc.ClientConn

	// Initiate a connection with the server
	conn, err := grpc.Dial("192.168.99.100:31250", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %s", err)
	}
	defer conn.Close()

	c := pb.NewDatacomClient(conn)

	// Test Execution
	_, err = c.Execute(context.Background(), &pb.ExecuteMessage{Execution: "foo"})
	if err != nil {
		t.Errorf("error when calling Execute: %s", err)
	}

	// Test Query
	response, err := c.Query(context.Background(), &pb.QueryMessage{Query: "foo"})
	if err != nil {
		t.Errorf("error when calling Execute: %s", err)
	}
	log.Println(response)
}
