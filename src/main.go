package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	. "github.com/olamai/proto/simulation"
	"google.golang.org/grpc"
)

type Server struct {
	world World
}

var (
	grpcport = flag.String("grpcport", ":50051", "grpcport")
	httpport = flag.String("httpport", ":80", "httpport")
)

// // Front handler
// func fronthandler(w http.ResponseWriter, r *http.Request) {
// 	//log.Println("Main Handler")
// 	fmt.Fprint(w, "hello world")
// }

// // Health handler for Ingress
// func healthhandler(w http.ResponseWriter, r *http.Request) {
// 	//log.Println("heathcheck...")
// 	fmt.Fprint(w, "ok")
// }

func main() {

	// Get flags
	flag.Parse()
	if *grpcport == "" {
		fmt.Fprintln(os.Stderr, "missing -grpcport flag (:50051)")
		flag.Usage()
		os.Exit(2)
	}
	// if *httpport == "" {
	// 	fmt.Fprintln(os.Stderr, "missing -httpport flag, using defaults(:8080)")
	// }

	// // Handle http
	// http.HandleFunc("/", fronthandler)
	// http.HandleFunc("/_ah/health", healthhandler)

	// // GRPC Server
	// srv := &http.Server{
	// 	Addr: *httpport,
	// }
	// http2.ConfigureServer(srv, &http2.Server{})
	// go srv.ListenAndServe()

	// Start listening
	lis, err := net.Listen("tcp", *grpcport)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	sopts := []grpc.ServerOption{grpc.MaxConcurrentStreams(10)}
	sopts = append(sopts)
	s := grpc.NewServer(sopts...)

	var simulationServer = Server{
		world: NewWorld(),
	}
	RegisterSimulationServer(s, &simulationServer)
	// healthpb.RegisterHealthServer(s, &health.Server{})

	log.Printf("Starting gRPC server on port %v", *grpcport)

	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
