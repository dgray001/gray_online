package lobby

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/fiddlesticks"
	"github.com/gin-gonic/gin"
)

type LobbyRoom struct {
	room_id       uint64
	room_name     string
	host          *Client
	players       map[uint64]*Client
	viewers       map[uint64]*Client
	lobby         *Lobby
	game          game.Game
	game_settings *GameSettings
}

type GameSettings struct {
	MaxPlayers uint8 `json:"max_players"`
	MaxViewers uint8 `json:"max_viewers"`
	GameType   uint8 `json:"game_type"`
}

func CreateLobbyRoom(host *Client, room_id uint64, lobby *Lobby) *LobbyRoom {
	room := LobbyRoom{
		room_id:   room_id,
		room_name: fmt.Sprintf("%s's room", host.nickname),
		host:      host,
		players:   make(map[uint64]*Client),
		viewers:   make(map[uint64]*Client),
		lobby:     lobby,
		game_settings: &GameSettings{
			MaxPlayers: 8,
			MaxViewers: 16,
			GameType:   0,
		},
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

func (r *LobbyRoom) setViewer(c *Client) {
	if r.host.client_id == c.client_id || r.players[c.client_id] == nil {
		return
	}
	delete(r.players, c.client_id)
	r.viewers[c.client_id] = c
	client_id_string := strconv.Itoa(int(c.client_id))
	room_id_string := strconv.Itoa(int(r.room_id))
	r.lobby.broadcastMessage(lobbyMessage{Sender: "room-" + room_id_string, Kind: "room-viewer-set", Data: client_id_string})
}

func (r *LobbyRoom) setPlayer(c *Client) {
	if r.host.client_id == c.client_id || r.viewers[c.client_id] == nil {
		return
	}
	delete(r.viewers, c.client_id)
	r.players[c.client_id] = c
	client_id_string := strconv.Itoa(int(c.client_id))
	room_id_string := strconv.Itoa(int(r.room_id))
	r.lobby.broadcastMessage(lobbyMessage{Sender: "room-" + room_id_string, Kind: "room-player-set", Data: client_id_string})
}

func (r *LobbyRoom) updateSettings(s *GameSettings) {
	settings_stringified, err := json.Marshal(s.ToFrontend())
	if err != nil {
		fmt.Println(err)
		return
	}
	r.game_settings = s
	room_id_string := strconv.Itoa(int(r.room_id))
	r.lobby.broadcastMessage(lobbyMessage{
		Sender: "room-" + room_id_string,
		Kind:   "room-settings-updated",
		Data:   string(settings_stringified),
	})
}

func (r *LobbyRoom) launchGame(game_id uint64) {
	if !r.launchable() {
		return
	}
	base_game := game.CreateBaseGame(game_id)
	for _, player := range r.players {
		base_game.Players[player.client_id] = &game.Player{}
	}
	for _, viewer := range r.viewers {
		base_game.Viewers[viewer.client_id] = &game.Viewer{}
	}
	switch r.game_settings.GameType {
	case 1:
		r.game = fiddlesticks.CreateGame(base_game)
	default:
		break
	}
	if r.game == nil {
		return
	}
	room_id_string := strconv.Itoa(int(r.room_id))
	game_id_string := strconv.Itoa(int(game_id))
	r.lobby.broadcastMessage(lobbyMessage{Sender: "room-" + room_id_string, Kind: "room-launched", Data: game_id_string})
}

func (s *GameSettings) Launchable() bool {
	if s.MaxPlayers < 1 || s.MaxPlayers > 8 {
		return false
	}
	if s.MaxViewers < 0 || s.MaxViewers > 16 {
		return false
	}
	if s.GameType != 1 {
		return false
	}
	return true
}

func (r *LobbyRoom) launchable() bool {
	if !r.valid() {
		return false
	}
	if !r.game_settings.Launchable() {
		return false
	}
	if r.game != nil {
		return false
	}
	return true
}

func (r *LobbyRoom) valid() bool {
	if r.room_id < 1 {
		return false
	}
	if r.host == nil || !r.host.valid() {
		return false
	}
	if len(r.players) > int(r.game_settings.MaxPlayers) {
		return false
	}
	if len(r.viewers) > int(r.game_settings.MaxViewers) {
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

func (s *GameSettings) ToFrontend() gin.H {
	return gin.H{
		"game_type":   strconv.Itoa(int(s.GameType)),
		"max_players": strconv.Itoa(int(s.MaxPlayers)),
		"max_viewers": strconv.Itoa(int(s.MaxViewers)),
	}
}

func (r *LobbyRoom) ToFrontend() gin.H {
	room := gin.H{
		"room_id":       strconv.Itoa(int(r.room_id)),
		"room_name":     r.room_name,
		"game_settings": r.game_settings.ToFrontend(),
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
	if r.game != nil {
		room["game_id"] = strconv.Itoa(int(r.game.GetId()))
	}
	return room
}
