package main

import (
	"log"
	"net"

	"github.com/lijuuu/ChallengeWssManagerService/internal/config"
	"github.com/lijuuu/ChallengeWssManagerService/internal/db"
	"github.com/lijuuu/ChallengeWssManagerService/internal/handlers"
	"github.com/lijuuu/ChallengeWssManagerService/internal/repo"
	"github.com/lijuuu/ChallengeWssManagerService/internal/service"
	challengepb "github.com/lijuuu/GlobalProtoXcode/ChallengeService"
	"google.golang.org/grpc"
)

const (
	grpcPort = ":50057"
	httpPort = ":8081"
)

func main() {
	//start gRPC server in a separate goroutine
	go runGRPCServer()

	//load configuration
	cfg := config.LoadConfig()

	//start HTTP server
	runHTTPServer(&cfg)
}

func runGRPCServer() {
	listener, err := net.Listen("tcp", grpcPort)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", grpcPort, err)
	}

	challengeSvc := service.NewChallengeService()
	grpcServer := grpc.NewServer()
	challengepb.RegisterChallengeServiceServer(grpcServer, challengeSvc)

	log.Printf("gRPC server listening on %s", grpcPort)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve gRPC: %v", err)
	}
}

func runHTTPServer(cfg *config.Config) {
	//initialize database
	dbInstance, err := db.InitDB(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize PostgreSQL DB: %v", err)
	}

	//initialize repository
	_ = repo.NewPSQLRepository(dbInstance)

	//pass to service

	//start HTTP server
	if err := handlers.StartServer(httpPort); err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}
