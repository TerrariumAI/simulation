package main

import (
	"flag"
	"log"
	"net"
	"os"

	api "github.com/terrariumai/simulation/pkg/api/collective"
	"github.com/terrariumai/simulation/pkg/collective"
	"google.golang.org/grpc"
)

// Config is configuration for Server
type Config struct {
	// gRPC server start parameters section
	// gRPC is TCP port to listen by gRPC server
	GRPCPort string
	// Environment service address
	EnvironmentAddr string
	// Environment that the server is running in (dev or prod)
	Env string
	// Log parameters section
	// LogLevel is global log level: Debug(-1), Info(0), Warn(1), Error(2), DPanic(3), Panic(4), Fatal(5)
	LogLevel int
	// LogTimeFormat is print time format for logger e.g. 2006-01-02T15:04:05Z07:00
	LogTimeFormat string
}

func main() {
	// get configuration
	var cfg Config
	flag.StringVar(&cfg.GRPCPort, "grpc-port", "", "gRPC port to bind")
	flag.StringVar(&cfg.EnvironmentAddr, "environment-addr", "", "Environment service address to connect to")
	flag.StringVar(&cfg.Env, "env", "", "Environment the server is running in")
	flag.IntVar(&cfg.LogLevel, "log-level", 0, "Global log level")
	flag.StringVar(&cfg.LogTimeFormat, "log-time-format", "",
		"Print time format for logger e.g. 2006-01-02T15:04:05Z07:00")
	flag.Parse()

	if len(cfg.GRPCPort) == 0 {
		log.Fatalf("invalid TCP port for gRPC server: '%s'", cfg.GRPCPort)
		os.Exit(1)
		return
	}
	if len(cfg.EnvironmentAddr) == 0 {
		log.Fatalf("invalid Environment address: '%s'", cfg.EnvironmentAddr)
		os.Exit(1)
		return
	}

	listen, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatalf("os.Stderr, '%v\n'", err)
		os.Exit(1)
	}

	serverAPI := collective.NewCollectiveServer(cfg.Env, cfg.EnvironmentAddr)

	opts := []grpc.ServerOption{}
	server := grpc.NewServer(opts...)
	api.RegisterCollectiveServer(server, serverAPI)

	log.Printf("Starting Collective Server on port %v", cfg.GRPCPort)
	server.Serve(listen)
}
