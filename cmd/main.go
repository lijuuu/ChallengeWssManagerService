package main

import (
	"log"
	"net"

	"github.com/lijuuu/ChallengeWssManagerService/internal/handlers"
	"github.com/lijuuu/ChallengeWssManagerService/internal/service"
	challengeService "github.com/lijuuu/GlobalProtoXcode/ChallengeService"
	"google.golang.org/grpc"
)

func startGRPCServer() {
	lis, err := net.Listen("tcp", ":50057")
	if err != nil {
		log.Fatalf("Failed to listen on port 50057: %v", err)
	}

	serviceInstance := service.NewChallengeService()

	grpcServer := grpc.NewServer()

	challengeService.RegisterChallengeServiceServer(grpcServer,serviceInstance)
	
	log.Println("gRPC server listening on :50057")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gRPC: %v", err)
	}
}

func main() {
	// Start gRPC server in a goroutine
	go startGRPCServer()

	// Start HTTP server
	addr := ":8080"
	if err := handlers.StartServer(addr); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
