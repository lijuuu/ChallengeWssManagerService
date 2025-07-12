package main

import (
	"log"
	// "math/rand"
	"net"
	"net/http"
	// "time"

	"github.com/lijuuu/ChallengeWssManagerService/internal/config"
	"github.com/lijuuu/ChallengeWssManagerService/internal/db"
	"github.com/lijuuu/ChallengeWssManagerService/internal/model"
	"github.com/lijuuu/ChallengeWssManagerService/internal/repo"
	"github.com/lijuuu/ChallengeWssManagerService/internal/service"
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

	// Initialize repository and service
	challengeRepo := repo.NewMongoRepository(mongoInstance, "challengeDB")
	challengeService := service.NewChallengeService(challengeRepo)

	// Start gRPC server in a goroutine
	go runGRPCServer(&cfg, challengeService)

	// Initialize WebSocket handler
	state := &wsstypes.State{
		Repo:       *challengeRepo,
		Challenges: make(map[string]*model.Challenge),
	}

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

	http.HandleFunc("/ws", wss.WsHandler(dispatcher, state))

	log.Println("Starting WebSocket server at ws://localhost:7777/ws")
	if err := http.ListenAndServe("0.0.0.0:7777", nil); err != nil {
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
