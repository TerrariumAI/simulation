package main

import (
	"io/ioutil"
	"log"
	"net"
	"os"

	"github.com/alicebob/miniredis/v2"
	collectiveApi "github.com/terrariumai/simulation/pkg/api/collective"
	envApi "github.com/terrariumai/simulation/pkg/api/environment"
	"github.com/terrariumai/simulation/pkg/collective"
	"github.com/terrariumai/simulation/pkg/console"
	"github.com/terrariumai/simulation/pkg/datacom"
	"github.com/terrariumai/simulation/pkg/environment"
	"google.golang.org/grpc"
)

func main() {
	// Disable logs because they mess with the console
	log.SetFlags(0)
	log.SetOutput(ioutil.Discard)

	// Create listeners
	cListen, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatalf("os.Stderr, '%v\n'", err)
		os.Exit(1)
	}
	eListen, err := net.Listen("tcp", ":9091")
	if err != nil {
		log.Fatalf("os.Stderr, '%v\n'", err)
		os.Exit(1)
	}

	// Start the redis service
	redisServer, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer redisServer.Close()

	// Create PAL (pubsub access layer) and DAL (data access layer)
	pubnubPAL := datacom.NewPubnubPAL("training", "", "")
	datacom, err := datacom.NewDatacom("training", redisServer.Addr(), pubnubPAL)

	// Create APIs
	cServerAPI := collective.NewCollectiveServer("training", redisServer.Addr(), "127.0.0.1:9091", pubnubPAL)
	eServerAPI := environment.NewEnvironmentServer("training", datacom)

	// Create servers
	opts := []grpc.ServerOption{}
	cServer := grpc.NewServer(opts...)
	collectiveApi.RegisterCollectiveServer(cServer, cServerAPI)
	eServer := grpc.NewServer(opts...)
	envApi.RegisterEnvironmentServer(eServer, eServerAPI)

	// Start 'em up!
	go eServer.Serve(eListen)
	defer eServer.Stop()
	go cServer.Serve(cListen)
	defer cServer.Stop()
	log.Println("Training environment is running locally on port 9090.")

	// Start the console to listen for commands
	console.StartConsole(eServerAPI)
}
