package lobby

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

type LobbyRoom struct {
	room_id uint64
	host    *Client
	clients map[uint64]*Client
}

func createRoom(host *Client, room_id uint64) *LobbyRoom {
	room := LobbyRoom{
		room_id: room_id,
		host:    host,
		clients: make(map[uint64]*Client),
	}
	room.clients[host.client_id] = host
	// TODO: host.send_message <- "host of new room: " + strconv.Itoa(int(room_id))
	return &room
}

func (r *LobbyRoom) valid() bool {
	if r.room_id < 1 {
		return false
	}
	if r.host == nil || !r.host.valid() {
		return false
	}
	return true
}

func (r *LobbyRoom) toFrontend() gin.H {
	return gin.H{
		"room_name": "id: " + strconv.Itoa(int(r.room_id)),
	}
}
