package lobby

import "github.com/gorilla/websocket"

type Lobby struct {
	rooms map[uint64]*LobbyRoom
}

type LobbyRoom struct {
	clients map[*Client]bool
}

type Client struct {
	client_id  uint64
	connection *websocket.Conn
}
