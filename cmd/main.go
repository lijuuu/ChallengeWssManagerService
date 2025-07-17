package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lijuuu/ChallengeWssManagerService/internal/config"
	"github.com/lijuuu/ChallengeWssManagerService/internal/db"
	"github.com/lijuuu/ChallengeWssManagerService/internal/repo"
	"github.com/lijuuu/ChallengeWssManagerService/internal/service"
	"github.com/lijuuu/ChallengeWssManagerService/internal/state"
	"github.com/lijuuu/ChallengeWssManagerService/internal/wss"
	"github.com/lijuuu/ChallengeWssManagerService/internal/wss/broadcasts"
	wsshandler "github.com/lijuuu/ChallengeWssManagerService/internal/wss/handlers"
	wsstypes "github.com/lijuuu/ChallengeWssManagerService/internal/wss/types"
	challengepb "github.com/lijuuu/GlobalProtoXcode/ChallengeService"
	"google.golang.org/grpc"
)

func main() {
	// Load config
	cfg := config.LoadConfig()
	// log.Printf("Loaded config: %+v", cfg)

	// Initialize MongoDB
	mongoInstance, err := db.InitDB(&cfg)
	if err != nil {
		log.Fatalf("Failed to initialize MongoDB: %v", err)

	}

	// Initialize Redis
	redisClient := db.NewRedisClient(cfg)

	// Load Redis data from RDB file on startup
	if err := db.LoadRedisData(redisClient); err != nil {
		log.Printf("Warning: Failed to load Redis data: %v", err)
	}

	// Initialize repositories
	challengeRepo := repo.NewMongoRepository(mongoInstance, "challengeDB")
	redisRepo := repo.NewRedisRepository(redisClient)

	// Initialize local state manager
	localStateManager := state.NewLocalStateManager()

	// Initialize leaderboard service
	leaderboardManager := service.NewLeaderboardManager(cfg.RedisURL, cfg.RedisPassword)

	// Initialize WebSocket state with both repositories and local state manager
	websocketState := &wsstypes.State{
		Redis:      redisRepo,
		Repo:       challengeRepo,
		LocalState: localStateManager,
	}

	// Initialize service with both repositories and WebSocket state
	challengeService := service.NewChallengeService(challengeRepo, redisRepo, websocketState)

	// Start gRPC server in a goroutine
	go runGRPCServer(&cfg, challengeService)

	dispatcher := wss.NewDispatcher()

	//ping for latency check
	dispatcher.Register(wsstypes.PING_SERVER, func(wc *wsstypes.WsContext) error {
		/*
			//to imitate irl latencies - use math/rand instead of crypto/rand
			rand.Seed(time.Now().UnixNano())
			randDuration := time.Duration(rand.Intn(1000)) * time.Millisecond
			time.Sleep(randDuration)*/
		return broadcasts.SendJSON(wc.Conn, map[string]interface{}{
			"type":    wsstypes.PING_SERVER,
			"status":  "ok",
			"message": "pong",
		})
	})

	//join challenge
	dispatcher.Register(wsstypes.JOIN_CHALLENGE, wsshandler.JoinChallengeHandler)

	//refetch challenge
	dispatcher.Register(wsstypes.REFETCH_CHALLENGE, wsshandler.RefetchChallenge)

	//get leaderboard
	dispatcher.Register(wsstypes.CURRENT_LEADERBOARD, wsshandler.NewGetLeaderboardHandler(leaderboardManager))

	http.HandleFunc("/ws", wss.WsHandler(dispatcher, websocketState))

	// Create HTTP server
	server := &http.Server{
		Addr: "0.0.0.0:7777",
	}

	// Setup graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("Shutting down gracefully...")

		// Save Redis data before shutdown
		if err := db.SaveRedisData(redisClient); err != nil {
			log.Printf("Error saving Redis data during shutdown: %v", err)
		}

		// Shutdown HTTP server with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}

		log.Println("Application shutdown complete")
		os.Exit(0)
	}()

	log.Println("Starting WebSocket server at ws://localhost:7777/ws")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("WebSocket server failed: %v", err)
	}
}

func runGRPCServer(cfg *config.Config, svc challengepb.ChallengeServiceServer) {
	addr := cfg.ChallengeGRPCPort
	if addr == "" {
		addr = ":50051"
	} else if addr[0] != ':' {
		addr = ":" + addr
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("gRPC server failed to listen on %s: %v", addr, err)
	}

	grpcServer := grpc.NewServer()
	challengepb.RegisterChallengeServiceServer(grpcServer, svc)

	log.Printf("Starting gRPC server at %s", addr)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("gRPC server failed to serve: %v", err)
	}
}
