package utils

import (
	"log"

	"github.com/gorilla/websocket"
	"github.com/lijuuu/ChallengeWssManagerService/internal/wss/broadcasts"
)

// CloseUnauthorizedConnection closes a WebSocket connection for unauthorized access
// and sends an error message before closing
func CloseUnauthorizedConnection(conn *websocket.Conn, messageType, errorMsg string, extra map[string]any) error {
	log.Printf("[WS] Closing unauthorized connection: %s", errorMsg)

	// Send error message before closing
	if err := broadcasts.SendErrorWithType(conn, messageType, errorMsg, extra); err != nil {
		log.Printf("[WS] Failed to send error before closing: %v", err)
	}

	// Close the connection
	if err := conn.Close(); err != nil {
		log.Printf("[WS] Failed to close connection: %v", err)
		return err
	}

	return nil
}

// CloseConnectionWithError closes a WebSocket connection and sends an error message
func CloseConnectionWithError(conn *websocket.Conn, messageType, errorMsg string) error {
	return CloseUnauthorizedConnection(conn, messageType, errorMsg, nil)
}
