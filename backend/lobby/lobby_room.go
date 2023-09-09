package lobby

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
)

type LobbyRoom struct {
	room_id   uint64
	room_name string
	host      *Client
	clients   map[uint64]*Client
	lobby     *Lobby
}

func CreateLobbyRoom(host *Client, room_id uint64, lobby *Lobby) *LobbyRoom {
	room := LobbyRoom{
		room_id:   room_id,
		room_name: fmt.Sprintf("%s's room", host.nickname),
		host:      host,
		clients:   make(map[uint64]*Client),
		lobby:     lobby,
	}
	room.clients[host.client_id] = host
	host.lobby_room = &room
	return &room
}

func (r *LobbyRoom) addClient(c *Client) {
	r.clients[c.client_id] = c
	if r.host.client_id == c.client_id {
		r.host = c
	}
	c.lobby_room = r
	client_id_string := strconv.Itoa(int(c.client_id))
	room_id_string := strconv.Itoa(int(r.room_id))
	r.lobby.broadcastMessage(lobbyMessage{Sender: "room-" + room_id_string, Kind: "room-joined", Data: client_id_string})
}

func (r *LobbyRoom) removeClient(c *Client) {
	if r.host.client_id == c.client_id {
		r.lobby.removeRoom(r)
		return
	}
	if c.lobby_room != nil && c.lobby_room.room_id == r.room_id {
		c.lobby_room = nil
	}
	delete(r.clients, c.client_id)
	client_id_string := strconv.Itoa(int(c.client_id))
	room_id_string := strconv.Itoa(int(r.room_id))
	r.lobby.broadcastMessage(lobbyMessage{Sender: "room-" + room_id_string, Kind: "room-left", Data: client_id_string})
}

func (r *LobbyRoom) kickClient(c *Client) {
	if r.host.client_id == c.client_id {
		return
	}
	if c.lobby_room != nil && c.lobby_room.room_id == r.room_id {
		c.lobby_room = nil
	}
	delete(r.clients, c.client_id)
	client_id_string := strconv.Itoa(int(c.client_id))
	room_id_string := strconv.Itoa(int(r.room_id))
	r.lobby.broadcastMessage(lobbyMessage{Sender: "room-" + room_id_string, Kind: "room-kicked", Data: client_id_string})
}

func (r *LobbyRoom) promoteClient(c *Client) {
	if r.host.client_id == c.client_id {
		return
	}
	if c.lobby_room == nil || c.lobby_room.room_id != r.room_id {
		return
	}
	r.host = c
	client_id_string := strconv.Itoa(int(c.client_id))
	room_id_string := strconv.Itoa(int(r.room_id))
	r.lobby.broadcastMessage(lobbyMessage{Sender: "room-" + room_id_string, Kind: "room-promoted", Data: client_id_string})
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
		"room_id":   strconv.Itoa(int(r.room_id)),
		"room_name": r.room_name,
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
