package lobby

import "github.com/gorilla/websocket"

type Client struct {
	client_id  uint64
	connection *websocket.Conn
}
