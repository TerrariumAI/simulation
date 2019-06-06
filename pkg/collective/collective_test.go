package collective

import (
	"context"
	"log"
	"testing"

	"google.golang.org/grpc"

	api "github.com/terrariumai/simulation/pkg/api/collective"
	"google.golang.org/grpc/metadata"
)

func TestConnectRemoteModel(t *testing.T) {
	conn, err := grpc.Dial("localhost:9090", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect: %s", err)
	}
	defer conn.Close()

	// ctxWithoutValidToken := context.Background()
	md := metadata.Pairs("auth-secret", "MOCK-SECRET", "model-name", "My Model")
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	c := api.NewCollectiveClient(conn)

	t.Run("Test connect RM", func(t *testing.T) {
		_, err := c.ConnectRemoteModel(ctx)
		if err != nil {
			t.Errorf("There was an error connecting: %v", err)
			return
		}
		println(err)
	})
}
