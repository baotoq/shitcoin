package ws

import (
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"
)

// upgrader upgrades HTTP connections to WebSocket.
// CheckOrigin allows all origins for development.
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// ServeWs returns an http.HandlerFunc that upgrades HTTP connections to WebSocket,
// creates a Client, registers it with the hub, and starts read/write pumps.
func ServeWs(hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			slog.Error("websocket upgrade failed", "error", err)
			return
		}

		client := &Client{
			hub:  hub,
			conn: conn,
			send: make(chan []byte, 256),
		}

		hub.register <- client

		go client.writePump()
		go client.readPump()
	}
}
