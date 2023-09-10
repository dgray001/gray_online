package lobby

import (
	"fmt"
	"strconv"

	"github.com/dgray001/gray_online/game"
	"github.com/gin-gonic/gin"
)

type LobbyRoom struct {
	room_id     uint64
	room_name   string
	host        *Client
	players     map[uint64]*Client
	viewers     map[uint64]*Client
	max_players uint8
	max_viewers uint8
	game_type   uint8
	lobby       *Lobby
	game        *game.Game
}

func CreateLobbyRoom(host *Client, room_id uint64, lobby *Lobby) *LobbyRoom {
	room := LobbyRoom{
		room_id:     room_id,
		room_name:   fmt.Sprintf("%s's room", host.nickname),
		host:        host,
		players:     make(map[uint64]*Client),
		viewers:     make(map[uint64]*Client),
		max_players: 8,
		max_viewers: 16,
		game_type:   0,
		lobby:       lobby,
	}
	room.players[host.client_id] = host
	host.lobby_room = &room
	return &room
}

func (r *LobbyRoom) addClient(c *Client) {
	r.players[c.client_id] = c
	if r.host.client_id == c.client_id {
		r.host = c
	}
	c.lobby_room = r
	client_id_string := strconv.Itoa(int(c.client_id))
	room_id_string := strconv.Itoa(int(r.room_id))
	r.lobby.broadcastMessage(lobbyMessage{Sender: "room-" + room_id_string, Kind: "room-joined-player", Data: client_id_string})
}

func (r *LobbyRoom) removeClient(c *Client) {
	if r.host.client_id == c.client_id {
		r.lobby.removeRoom(r)
		return
	}
	if c.lobby_room != nil && c.lobby_room.room_id == r.room_id {
		c.lobby_room = nil
	}
	delete(r.players, c.client_id)
	delete(r.viewers, c.client_id)
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
	delete(r.players, c.client_id)
	delete(r.viewers, c.client_id)
	client_id_string := strconv.Itoa(int(c.client_id))
	room_id_string := strconv.Itoa(int(r.room_id))
	r.lobby.broadcastMessage(lobbyMessage{Sender: "room-" + room_id_string, Kind: "room-kicked", Data: client_id_string})
}

func (r *LobbyRoom) promotePlayer(c *Client) {
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
	for _, client := range r.players {
		if client == nil || !client.valid() {
			return false
		}
	}
	for _, client := range r.viewers {
		if client == nil || !client.valid() {
			return false
		}
	}
	return true
}

func (r *LobbyRoom) toFrontend() gin.H {
	room := gin.H{
		"room_id":     strconv.Itoa(int(r.room_id)),
		"room_name":   r.room_name,
		"game_type":   strconv.Itoa(int(r.game_type)),
		"max_players": strconv.Itoa(int(r.max_players)),
		"max_viewers": strconv.Itoa(int(r.max_viewers)),
	}
	if r.host != nil {
		room["host"] = r.host.toFrontend()
	}
	players := []gin.H{}
	for _, player := range r.players {
		if player != nil {
			players = append(players, player.toFrontend())
		}
	}
	room["players"] = players
	viewers := []gin.H{}
	for _, viewer := range r.viewers {
		if viewer != nil {
			viewers = append(viewers, viewer.toFrontend())
		}
	}
	room["viewers"] = viewers
	return room
}
