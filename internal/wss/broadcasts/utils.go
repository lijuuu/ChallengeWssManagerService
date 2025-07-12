package broadcasts

import (
	"github.com/gorilla/websocket"
)

func SendJSON(conn *websocket.Conn, data interface{}) error {
	return conn.WriteJSON(data)
}
func SendErrorWithType(conn *websocket.Conn, eventType string, msg string, extra map[string]interface{}) error {
	response := map[string]interface{}{
		"type":    eventType,
		"status":  "error",
		"message": msg,
	}
	for k, v := range extra {
		response[k] = v
	}
	return SendJSON(conn, response)
}
