package main

import (
	"log"

	"github.com/lijuuu/ChallengeWssManagerService/internal/handlers"
)

func main() {
	addr := ":8080"
	if err := handlers.StartServer(addr); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
