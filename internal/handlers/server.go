package handlers

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

// StartServer starts the game manager HTTP server
func StartServer(addr string) error {
	r := mux.NewRouter()

	// WebSocket endpoint
	r.HandleFunc("/ws/{challenge_id}", WebSocketHandler).Methods("GET")

	// REST API endpoints
	r.HandleFunc("/challenges", CreateChallengeHandler).Methods("POST")
	r.HandleFunc("/challenges", ListChallengesHandler).Methods("GET")
	r.HandleFunc("/challenges/{challenge_id}", GetChallengeHandler).Methods("GET")

	http.Handle("/", r)
	log.Printf("Starting game manager server on %s", addr)
	return http.ListenAndServe(addr, nil)
}
