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

func CreateLobbyRoom(host *Client, room_id uint64) *LobbyRoom {
	room := LobbyRoom{
		room_id: room_id,
		host:    host,
		clients: make(map[uint64]*Client),
	}
	room.clients[host.client_id] = host
	return &room
}

func (r *LobbyRoom) valid() bool {
	if r.room_id < 1 {
		return false
	}
	if r.host == nil || !r.host.valid() {
		return false
	}
	for _, client := range r.clients {
		if client == nil || !client.valid() {
			return false
		}
	}
	return true
}

func (r *LobbyRoom) toFrontend() gin.H {
	room := gin.H{
		"room_id": strconv.Itoa(int(r.room_id)),
	}
	if r.host != nil {
		room["host"] = r.host.toFrontend()
	}
	users := []gin.H{}
	for _, user := range r.clients {
		if user != nil {
			users = append(users, user.toFrontend())
		}
	}
	room["users"] = users
	return room
}
