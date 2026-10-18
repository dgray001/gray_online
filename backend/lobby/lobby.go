package lobby

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/util"
	"github.com/gin-gonic/gin"
)

// How long a disconnected client keeps its room spot (outside a running game seat) to allow a reconnect
const room_rejoin_grace = 5 * time.Second

type Lobby struct {
	std                   *util.Logger
	err                   *util.Logger
	mu                    sync.Mutex
	next_room_id          uint64
	next_client_id        uint64
	next_game_id          uint64
	clients               map[uint64]*Client
	rooms                 map[uint64]*LobbyRoom
	games                 map[uint64]game.Game
	AddClient             chan *Client
	ReconnectClient       chan *ClientId
	RemoveClient          chan *Client
	CreateRoom            chan *Client
	RenameRoom            chan *LobbyRoom
	UpdateRoomDescription chan *LobbyRoom
	KickClientFromRoom    chan *ClientRoom
	PromotePlayerInRoom   chan *ClientRoom
	RoomSetViewer         chan *ClientRoom
	RoomSetPlayer         chan *ClientRoom
	RemoveRoom            chan *LobbyRoom
	LeaveRoom             chan *ClientRoom
	LaunchGame            chan *LobbyRoom
	broadcast             chan lobbyMessage
}

type ClientRoom struct {
	client    *Client
	room      *LobbyRoom
	bool_flag bool
}

func MakeClientRoom(client *Client, room *LobbyRoom) *ClientRoom {
	return &ClientRoom{
		client: client,
		room:   room,
	}
}

type ClientId struct {
	client    *Client
	client_id uint64
}

func MakeClientId(client *Client, client_id uint64) *ClientId {
	return &ClientId{
		client:    client,
		client_id: client_id,
	}
}

func CreateLobby(std, err *util.Logger) *Lobby {
	if std == nil {
		std = util.DefaultStd
	}
	if err == nil {
		err = util.DefaultErr
	}
	return &Lobby{
		std:                   std,
		err:                   err,
		next_room_id:          1,
		next_client_id:        1,
		next_game_id:          1,
		clients:               make(map[uint64]*Client),
		rooms:                 make(map[uint64]*LobbyRoom),
		games:                 make(map[uint64]game.Game),
		AddClient:             make(chan *Client, 4),
		ReconnectClient:       make(chan *ClientId, 4),
		RemoveClient:          make(chan *Client, 4),
		CreateRoom:            make(chan *Client, 4),
		RenameRoom:            make(chan *LobbyRoom, 4),
		UpdateRoomDescription: make(chan *LobbyRoom, 4),
		KickClientFromRoom:    make(chan *ClientRoom, 4),
		PromotePlayerInRoom:   make(chan *ClientRoom, 4),
		RoomSetViewer:         make(chan *ClientRoom, 4),
		RoomSetPlayer:         make(chan *ClientRoom, 4),
		RemoveRoom:            make(chan *LobbyRoom, 4),
		LeaveRoom:             make(chan *ClientRoom, 4),
		LaunchGame:            make(chan *LobbyRoom, 4),
		broadcast:             make(chan lobbyMessage, 24),
	}
}

func (l *Lobby) Run() {
	for {
		l.runOnce()
	}
}

func (l *Lobby) runOnce() {
	defer func() {
		if rec := recover(); rec != nil {
			util.Errorf(l.err, "recovered from panic in lobby actor loop: %v", rec)
		}
	}()
	select {
	case client := <-l.AddClient:
		l.addClient(client)
	case client_id := <-l.ReconnectClient:
		l.reconnectClient(client_id.client, client_id.client_id)
	case client := <-l.RemoveClient:
		l.removeClient(client)
	case client := <-l.CreateRoom:
		l.createRoom(client)
	case room := <-l.RenameRoom:
		l.renameRoom(room)
	case room := <-l.UpdateRoomDescription:
		l.updateRoomDescription(room)
	case data := <-l.KickClientFromRoom:
		data.room.kickClient(data.client)
	case data := <-l.PromotePlayerInRoom:
		data.room.promotePlayer(data.client)
	case data := <-l.RoomSetViewer:
		data.room.setViewer(data.client)
	case data := <-l.RoomSetPlayer:
		data.room.setPlayer(data.client)
	case room := <-l.RemoveRoom:
		l.removeRoom(room)
	case data := <-l.LeaveRoom:
		l.leaveRoom(data)
	case room := <-l.LaunchGame:
		game_id := l.next_game_id
		game, err := room.launchGame(game_id)
		if err == nil {
			l.mu.Lock()
			l.games[game_id] = game
			l.next_game_id++
			l.mu.Unlock()
			room_id_string := strconv.Itoa(int(room.room_id))
			game_id_string := strconv.Itoa(int(game_id))
			l.broadcastMessage(lobbyMessage{Sender: "room-" + room_id_string, Kind: "room-launched", Data: game_id_string})
		} else if room.host != nil {
			room.host.send(lobbyMessage{Sender: "server", Kind: "room-launch-failed", Content: err.Error()})
		}
	case message := <-l.broadcast:
		l.broadcastMessage(message)
	}
}

func (l *Lobby) GetRooms() []gin.H {
	l.mu.Lock()
	defer l.mu.Unlock()
	rooms := []gin.H{}
	for _, room := range l.rooms {
		if room.validLocked() {
			rooms = append(rooms, room.toFrontendLocked())
		}
	}
	return rooms
}

func (l *Lobby) GetUsers() []gin.H {
	l.mu.Lock()
	defer l.mu.Unlock()
	users := []gin.H{}
	for _, user := range l.clients {
		if user.validLocked() {
			users = append(users, user.toFrontendLocked())
		}
	}
	return users
}

func (l *Lobby) GetGames() []gin.H {
	l.mu.Lock()
	defer l.mu.Unlock()
	games := []gin.H{}
	for _, room := range l.rooms {
		if room.validLocked() {
			game := room.game
			if game != nil && game.Valid() {
				games = append(games, game.ToFrontend(0, false))
			}
		}
	}
	return games
}

func (l *Lobby) GetClient(client_id uint64) *Client {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.clients[client_id]
}

func (l *Lobby) GetRoom(room_id uint64) *LobbyRoom {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.rooms[room_id]
}

func (l *Lobby) GetGame(game_id uint64) game.Game {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.games[game_id]
}

func (l *Lobby) addClient(client *Client) {
	l.mu.Lock()
	l.setClientId(client)
	l.clients[client.client_id] = client
	l.mu.Unlock()
	id_string := strconv.Itoa(int(client.client_id))
	client.send(lobbyMessage{Sender: "server", Kind: "lobby-you-joined", Content: client.nickname, Data: id_string + "-connect-" + client.token})
	l.broadcastMessage(lobbyMessage{Sender: "client-" + id_string, Kind: "lobby-joined", Content: client.nickname, Data: id_string + "-connect"})
}

func (l *Lobby) reconnectClient(client *Client, client_id uint64) {
	l.mu.Lock()
	old_client := l.clients[client_id]
	if old_client == nil {
		l.mu.Unlock()
		util.Info(l.std, "Cannot find old client; starting new connection")
		l.addClient(client)
		return
	}
	util.Infof(l.std, "Reconnecting to old client %d (old connection still valid: %t)", old_client.client_id, old_client.validLocked())
	if old_client.delete_timer != nil {
		old_client.delete_timer.Stop()
	}
	close(old_client.quit)
	old_connection := old_client.connection
	client.client_id = old_client.client_id
	old_game := old_client.game
	old_lobby_room := old_client.lobby_room
	if old_client.gameNil() && old_lobby_room != nil {
		old_game = old_lobby_room.game
	}
	l.mu.Unlock()
	go closeReplacedConnection(old_connection)
	if old_lobby_room != nil {
		if old_game == nil || old_game.GetBase() == nil {
			util.Infof(l.std, "Reconnecting client %d to room %d", client.client_id, old_lobby_room.room_id)
			client.lobby_room = old_lobby_room
			old_lobby_room.PlayerReplaced <- client
		} else {
			util.Infof(l.std, "Reconnecting to game %d in room %d with old client %d", old_game.GetBase().Game_id, old_lobby_room.room_id, client.client_id)
			client.lobby_room = old_lobby_room
			client.game = old_game
			room_id_string := strconv.Itoa(int(client.lobby_room.room_id))
			old_lobby_room.PlayerReplaced <- client
			player := client.game.GetBase().Players[client_id]
			if player == nil {
				viewer := client.game.GetBase().Viewers[client_id]
				if viewer == nil {
					util.Error(l.err, "Room doesn't have reconnecting client")
					client.lobby_room = nil
					client.game = nil
				} else {
					util.Info(l.std, "Starting reconnect viewer game updates")
					go client.viewerGameUpdates(viewer, room_id_string)
				}
			} else {
				util.Infof(l.std, "Starting reconnect player game updates for player %d", player.Player_id)
				go client.playerGameUpdates(player, room_id_string)
			}
		}
	}
	l.mu.Lock()
	l.clients[client.client_id] = client
	old_client.connection = nil
	old_client.deleted = true
	l.mu.Unlock()
	id_string := strconv.Itoa(int(client.client_id))
	client.send(lobbyMessage{Sender: "server", Kind: "lobby-you-joined", Content: client.nickname, Data: id_string + "-reconnect-" + client.token})
	l.broadcastMessage(lobbyMessage{Sender: "client-" + id_string, Kind: "lobby-joined", Content: client.nickname, Data: id_string + "-reconnect"})
}

func (l *Lobby) removeClient(client *Client) {
	l.mu.Lock()
	if l.clients[client.client_id] != client {
		l.mu.Unlock()
		return // this client_id was already taken over by a reconnect
	}
	delete_client := time.NewTimer(1 * time.Hour)
	client.delete_timer = delete_client
	lobby_room := client.lobby_room
	holds_game_seat := lobby_room != nil && lobby_room.gameStartedLocked() && lobby_room.players[client.client_id] == client
	l.mu.Unlock()
	go func() {
		<-delete_client.C
		l.mu.Lock()
		client.deleted = true
		delete(l.clients, client.client_id)
		l.mu.Unlock()
	}()
	id_string := strconv.Itoa(int(client.client_id))
	if holds_game_seat {
		lobby_room.LeaveRoom <- &ClientRoom{client: client, room: lobby_room}
	} else if lobby_room != nil {
		time.AfterFunc(room_rejoin_grace, func() {
			if l.GetClient(client.client_id) == client {
				lobby_room.LeaveRoom <- &ClientRoom{client: client, room: lobby_room}
			}
		})
	}
	l.broadcastMessage(lobbyMessage{Sender: "client-" + id_string, Kind: "lobby-left", Content: client.nickname, Data: id_string})
}

func (l *Lobby) createRoom(client *Client) {
	if client.lobby_room != nil {
		client.lobby_room.LeaveRoom <- &ClientRoom{client: client, room: client.lobby_room}
	}
	l.mu.Lock()
	room_id := l.next_room_id
	l.next_room_id++
	room := CreateLobbyRoom(client, room_id, l)
	l.rooms[room.room_id] = room
	l.mu.Unlock()
	id_string := strconv.Itoa(int(room.room_id))
	client_id_string := strconv.Itoa(int(client.client_id))
	room_stringified, err := json.Marshal(room.ToFrontend())
	if err != nil {
		util.Error(l.err, err.Error())
		room_stringified = []byte{}
	}
	l.broadcastMessage(lobbyMessage{
		Sender:  "client-" + client_id_string,
		Kind:    "room-created",
		Content: string(room_stringified),
		Data:    id_string,
	})
}

func (l *Lobby) renameRoom(room *LobbyRoom) {
	l.mu.Lock()
	if room == nil || room.host == nil {
		l.mu.Unlock()
		return
	}
	room_id_string := strconv.Itoa(int(room.room_id))
	host_id_string := strconv.Itoa(int(room.host.client_id))
	room_name := room.room_name
	l.mu.Unlock()
	l.broadcastMessage(lobbyMessage{
		Sender:  "room-" + room_id_string,
		Kind:    "room-renamed",
		Content: room_name,
		Data:    host_id_string,
	})
}

func (l *Lobby) updateRoomDescription(room *LobbyRoom) {
	l.mu.Lock()
	if room == nil || room.host == nil {
		l.mu.Unlock()
		return
	}
	room_id_string := strconv.Itoa(int(room.room_id))
	host_id_string := strconv.Itoa(int(room.host.client_id))
	room_description := room.room_description
	l.mu.Unlock()
	l.broadcastMessage(lobbyMessage{
		Sender:  "room-" + room_id_string,
		Kind:    "room-description-updated",
		Content: room_description,
		Data:    host_id_string,
	})
}

func (l *Lobby) removeRoom(room *LobbyRoom) {
	l.mu.Lock()
	delete(l.rooms, room.room_id)
	id_string := strconv.Itoa(int(room.room_id))
	if room.host != nil {
		room.host.lobby_room = nil
	}
	for _, player := range room.players {
		if player != nil {
			player.lobby_room = nil
		}
	}
	for _, viewer := range room.viewers {
		if viewer != nil {
			viewer.lobby_room = nil
		}
	}
	l.mu.Unlock()
	l.broadcastMessage(lobbyMessage{Sender: "server", Kind: "room-closed", Data: id_string})
}

func (l *Lobby) leaveRoom(data *ClientRoom) {
	data.room.LeaveRoom <- &ClientRoom{client: data.client, room: data.room, bool_flag: true}
}

var (
	client_to_lobby_messages = []string{"lobby-joined", "lobby-left", "lobby-chat"}
	lobby_messages           = []string{
		"room-created", "room-closed", "room-joined-player", "room-joined-viewer",
		"room-left", "room-renamed", "room-kicked", "room-promoted", "ping-update",
		"room-settings-updated", "room-viewer-set", "room-player-set", "room-launched",
		"room-game-over", "room-description-updated",
	}
	client_to_room_messages = []string{"room-chat"}
	room_messages           = []string{}
)

func (l *Lobby) broadcastMessage(message lobbyMessage) {
	if message.Kind != "ping-update" {
		util.Infof(l.std, "Broadcasting message {%s, %s, %s, %s}", message.Sender, message.Content, message.Data, message.Kind)
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if util.Contains(client_to_lobby_messages, message.Kind) {
		send_id_string := strings.TrimPrefix(message.Sender, "client-")
		sender_id, err := strconv.ParseInt(send_id_string, 10, 0)
		if err != nil {
			sender_id = -1
		}
		for _, client := range l.clients {
			if client == nil || !client.validLocked() || client.client_id == uint64(sender_id) {
				continue
			}
			client.send(message)
		}
	} else if util.Contains(lobby_messages, message.Kind) {
		for _, client := range l.clients {
			if client == nil || !client.validLocked() {
				continue
			}
			client.send(message)
		}
	} else if util.Contains(client_to_room_messages, message.Kind) {
		send_split := strings.Split(message.Sender, "-")
		if len(send_split) < 3 {
			util.Error(l.err, "Sender not properly formed for a client to room message")
			return
		}
		room_id, err := strconv.ParseInt(send_split[1], 10, 0)
		if err != nil {
			room_id = -1
		}
		client_id, err := strconv.ParseInt(send_split[2], 10, 0)
		if err != nil {
			client_id = -1
		}
		util.Infof(l.std, "Sending chat from client %d", client_id)
		for _, client := range l.clients {
			if client == nil || !client.validLocked() || client.lobby_room == nil ||
				client.lobby_room.room_id != uint64(room_id) || client.client_id == uint64(client_id) {
				continue
			}
			client.send(message)
		}
	} else if util.Contains(room_messages, message.Kind) {
		room_id_string := strings.TrimPrefix(message.Sender, "room-")
		room_id, err := strconv.ParseInt(room_id_string, 10, 0)
		if err != nil {
			room_id = -1
		}
		for _, client := range l.clients {
			if client == nil || !client.validLocked() || client.lobby_room == nil || client.lobby_room.room_id != uint64(room_id) {
				continue
			}
			client.send(message)
		}
	} else {
		util.Error(l.err, "No logic for message kind")
	}
}

func (l *Lobby) setClientId(client *Client) {
	if client.client_id > 0 {
		util.Infof(l.std, "Client already has id: %d", client.client_id)
		return
	}
	client.client_id = l.next_client_id
	l.next_client_id++
}
