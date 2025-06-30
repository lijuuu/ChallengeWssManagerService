package main

import (
	"log"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lijuuu/ChallengeWssManagerService/internal/config"
	"github.com/lijuuu/ChallengeWssManagerService/internal/db"

	// "github.com/lijuuu/ChallengeWssManagerService/internal/http"
	"github.com/lijuuu/ChallengeWssManagerService/internal/repo"
	"github.com/lijuuu/ChallengeWssManagerService/internal/service"
	challengepb "github.com/lijuuu/GlobalProtoXcode/ChallengeService"
	"google.golang.org/grpc"
)

func main() {
	// Load config
	cfg := config.LoadConfig()

	// Init DB
	mongoInstance, err := db.InitDB(&cfg)
	if err != nil {
		log.Fatalf("Failed to init PostgreSQL: %v", err)
	}

	// Init repo and service
	challengeRepo := repo.NewMongoRepository(mongoInstance,"challengeDB")
	challengeService := service.NewChallengeService(challengeRepo)

	// gRPC in goroutine
	go runGRPCServer(&cfg, challengeService)

	// HTTP using Gin
	runHTTPServer(&cfg, challengeService)
}

func runGRPCServer(cfg *config.Config, svc challengepb.ChallengeServiceServer) {
	addr := cfg.ChallengeGRPCPort
	if addr[0] != ':' {
		addr = ":" + addr
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("gRPC listen failed: %v", err)
	}

	grpcServer := grpc.NewServer()
	challengepb.RegisterChallengeServiceServer(grpcServer, svc)

	log.Printf("gRPC server on %s", cfg.ChallengeGRPCPort)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("gRPC serve error: %v", err)
	}
}

func runHTTPServer(cfg *config.Config, svc *service.ChallengeService) {
	r := gin.Default()
	// http.RegisterRoutes(r, svc)

	addr := cfg.ChallengeHTTPPort
	if addr[0] != ':' {
		addr = ":" + addr
	}

	log.Printf("HTTP server on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("HTTP serve error: %v", err)
	}
}
