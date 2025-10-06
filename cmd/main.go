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
	"github.com/lijuuu/ChallengeWssManagerService/internal/constants"
	"github.com/lijuuu/ChallengeWssManagerService/internal/db"
	"github.com/lijuuu/ChallengeWssManagerService/internal/global"
	"github.com/lijuuu/ChallengeWssManagerService/internal/jwt"
	"github.com/lijuuu/ChallengeWssManagerService/internal/leaderboard"
	localstate "github.com/lijuuu/ChallengeWssManagerService/internal/local"
	"github.com/lijuuu/ChallengeWssManagerService/internal/repo"
	"github.com/lijuuu/ChallengeWssManagerService/internal/service"
	"github.com/lijuuu/ChallengeWssManagerService/internal/wss"
	"github.com/lijuuu/ChallengeWssManagerService/internal/wss/broadcasts"
	wsshandler "github.com/lijuuu/ChallengeWssManagerService/internal/wss/handlers"
	wsstypes "github.com/lijuuu/ChallengeWssManagerService/internal/wss/types"
	challengepb "github.com/lijuuu/GlobalProtoXcode/ChallengeService"
	"github.com/panjf2000/ants/v2"
	"google.golang.org/grpc"
)

func main() {
	//load environment configuration.
	cfg := config.LoadConfig()
	// log.Printf("Loaded config: %+v", cfg)

	//set up mongodb.
	mongoInstance, err := db.InitDB(&cfg)
	if err != nil {
		log.Fatalf("Failed to initialize MongoDB: %v", err)

	}

	//set up redis.
	redisClient := db.NewRedisClient(cfg)

	//on startup, attempt to restore data from redis rdb.
	if err := db.LoadRedisData(redisClient); err != nil {
		log.Printf("Warning: Failed to load Redis data: %v", err)
	}

	jwtManager := jwt.NewJWTManager(config.LoadConfig().JWTSecret)

	//create repositories that wrap mongodb and redis.
	mongoRepo := repo.NewMongoRepository(mongoInstance, "challengeDB")
	redisRepo := repo.NewRedisRepository(redisClient)

	//in-memory state for websocket sessions.
	localStateManager := localstate.NewLocalStateManager()

	//ranking/leaderboard manager backed by redis.
	leaderboardManager := leaderboard.NewLeaderboardManager(cfg.RedisURL, cfg.RedisPassword)

	//worker pool for background tasks.
	pool, _ := ants.NewPool(100)
	defer pool.Release()

	//bundle all shared dependencies.
	globalState := &global.State{
		Redis:               redisRepo,
		Mongo:               mongoRepo,
		LocalState:          localStateManager,
		JwtManager:          jwtManager,
		LeaderboardManager:  leaderboardManager,
		AntsWorkerPool:      pool,
		ChallengeSchedulers: make(map[string]*time.Timer),
	}

	
	//grpc service implementation using shared state.
	challengeService := service.NewChallengeService(globalState)
	
	//on server restarts check redis and update whole active challenge's schedulers.
	challengeService.WarmUpScheduler(context.Background())
	//start the grpc server alongside the websocket server.
	go runGRPCServer(&cfg, challengeService)

	dispatcher := wss.NewDispatcher()

	//jwt guard for routes that require an authenticated challenge session.
	jwtMiddleware := func(ctx *wsstypes.WsContext) error {
		//read token from the incoming payload.
		var token string
		if tokenVal, exists := ctx.Payload["challengeToken"]; exists {
			if tokenStr, ok := tokenVal.(string); ok {
				token = tokenStr
			}
		}

		if token == "" {
			return broadcasts.SendErrorWithType(ctx.Conn, "AUTH_ERROR", "Authentication token required", nil)
		}

		//validate the token and attach claims to the context.
		claims, err := jwtManager.ValidateToken(token)
		if err != nil {
			return broadcasts.SendErrorWithType(ctx.Conn, "AUTH_ERROR", "Invalid or expired token", nil)
		}

		//make claims available to downstream handlers.
		ctx.Claims = claims
		ctx.UserID = claims.UserID

		return nil
	}

	//health check: simple ping without auth.
	dispatcher.Register(wsstypes.PING_SERVER, func(wc *wsstypes.WsContext) error {
		return broadcasts.SendJSON(wc.Conn, map[string]interface{}{
			"type":    wsstypes.PING_SERVER,
			"status":  "ok",
			"message": "pong",
		})
	})

	//join a challenge (issues a token). no prior auth required.
	dispatcher.Register(wsstypes.JOIN_CHALLENGE, wsshandler.JoinChallengeHandler)

	//refetch challenge details (requires a valid token).
	dispatcher.RegisterWithMiddleware(wsstypes.RETRIEVE_CHALLENGE, wsshandler.RetreiveChallenge, jwtMiddleware)

	//fetch current leaderboard (requires a valid token).
	dispatcher.RegisterWithMiddleware(wsstypes.CURRENT_LEADERBOARD, wsshandler.GetLeaderboardHandler, jwtMiddleware)

	//bff style data fetchers (requires a valid token).
	dispatcher.RegisterWithMiddleware(constants.GET_CHALLENGE_DATA, wsshandler.GetChallengeDataHandler, jwtMiddleware)
	dispatcher.RegisterWithMiddleware(constants.GET_CHALLENGE_MIN, wsshandler.GetChallengeMinHandler, jwtMiddleware)
	dispatcher.RegisterWithMiddleware(constants.GET_PARTICIPANT_DATA, wsshandler.GetParticipantDataHandler, jwtMiddleware)
	dispatcher.RegisterWithMiddleware(constants.GET_PARTICIPANTS_DATA, wsshandler.GetParticipantsDataHandler, jwtMiddleware)
	dispatcher.RegisterWithMiddleware(constants.WHOLE_NOTIFICATION, wsshandler.GetNotificationsHandler, jwtMiddleware)
	dispatcher.RegisterWithMiddleware(constants.WHOLE_CHAT, wsshandler.GetChatHandler, jwtMiddleware)
	dispatcher.RegisterWithMiddleware(constants.PUSH_NEW_CHAT, wsshandler.PushNewChatHandler, jwtMiddleware)
	dispatcher.RegisterWithMiddleware(constants.PUSH_NEW_CHAT, wsshandler.PushNewChatSyncHandler, jwtMiddleware)
	dispatcher.RegisterWithMiddleware(constants.PUSH_NEW_NOTIFICATION, wsshandler.PushNewNotificationHandler, jwtMiddleware)

	http.HandleFunc("/ws", wss.WsHandler(dispatcher, globalState))

	//http server hosting the websocket endpoint.
	server := &http.Server{
		Addr: "0.0.0.0:7777",
	}

	//graceful shutdown on sigterm/sigint.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("Shutting down gracefully...")

		//persist redis state to disk before exiting.
		if err := db.SaveRedisData(redisClient); err != nil {
			log.Printf("Error saving Redis data during shutdown: %v", err)
		}

		//stop the http server with a timeout.
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
