package lobby

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/util"
	"github.com/gin-gonic/gin"
)

type Lobby struct {
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

func CreateLobby() *Lobby {
	return &Lobby{
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
			game, err := room.launchGame(l.next_game_id)
			if err == nil {
				l.games[l.next_game_id] = game
				l.next_game_id++
			} else if room.host != nil {
				room.host.send_message <- lobbyMessage{Sender: "server", Kind: "room-launch-failed", Content: err.Error()}
			}
		case message := <-l.broadcast:
			l.broadcastMessage(message)
		}
	}
}

func (l *Lobby) GetRooms() []gin.H {
	rooms := []gin.H{}
	for _, room := range l.rooms {
		if room.valid() {
			rooms = append(rooms, room.ToFrontend())
		}
	}
	return rooms
}

func (l *Lobby) GetUsers() []gin.H {
	users := []gin.H{}
	for _, user := range l.clients {
		if user.valid() {
			users = append(users, user.ToFrontend())
		}
	}
	return users
}

func (l *Lobby) GetGames() []gin.H {
	games := []gin.H{}
	for _, room := range l.rooms {
		if room.valid() {
			game := room.game
			if game != nil && game.Valid() {
				games = append(games, game.ToFrontend(0, false))
			}
		}
	}
	return games
}

func (l *Lobby) GetClient(client_id uint64) *Client {
	return l.clients[client_id]
}

func (l *Lobby) GetRoom(room_id uint64) *LobbyRoom {
	return l.rooms[room_id]
}

func (l *Lobby) GetGame(game_id uint64) game.Game {
	return l.games[game_id]
}

func (l *Lobby) addClient(client *Client) {
	l.setClientId(client)
	l.clients[client.client_id] = client
	id_string := strconv.Itoa(int(client.client_id))
	client.send_message <- lobbyMessage{Sender: "server", Kind: "lobby-you-joined", Content: client.nickname, Data: id_string + "-connect"}
	l.broadcastMessage(lobbyMessage{Sender: "client-" + id_string, Kind: "lobby-joined", Content: client.nickname, Data: id_string + "-connect"})
}

func (l *Lobby) reconnectClient(client *Client, client_id uint64) {
	old_client := l.clients[client_id]
	if old_client == nil {
		fmt.Println("Cannot find old client; starting new connection")
		l.addClient(client)
		return
	}
	if old_client.valid() {
		fmt.Println("Old client is valid; starting new connection")
		l.addClient(client)
		return
	}
	fmt.Println("Reconnecting to old client:", old_client.client_id)
	if old_client.delete_timer != nil {
		old_client.delete_timer.Stop()
	}
	client.client_id = old_client.client_id
	old_game := old_client.game
	old_lobby_room := old_client.lobby_room
	if old_client.gameNil() && old_lobby_room != nil {
		old_game = old_lobby_room.game
	}
	if old_lobby_room != nil {
		if old_game == nil || old_game.GetBase() == nil {
			fmt.Println("Removing client", client.client_id, "from old room", old_lobby_room.room_id)
			old_lobby_room.removeClient(client, true)
		} else {
			fmt.Println("Reconnecting to game", old_game.GetBase().Game_id, "in room", old_lobby_room.room_id, "with old client", client.client_id)
			client.lobby_room = old_lobby_room
			client.game = old_game
			room_id_string := strconv.Itoa(int(client.lobby_room.room_id))
			old_lobby_room.replaceClient(client) // doesn't return if room has client
			player := client.game.GetBase().Players[client_id]
			if player == nil {
				viewer := client.game.GetBase().Viewers[client_id]
				if viewer == nil {
					fmt.Fprintln(os.Stderr, "Room doesn't have reconnecting client")
					client.lobby_room = nil
					client.game = nil
				} else {
					fmt.Println("Starting reconnect viewer game updates")
					go client.viewerGameUpdates(viewer, room_id_string)
				}
			} else {
				fmt.Println("Starting reconnect player game updates for player", player.Player_id)
				player.FlushConnections <- false
				go client.playerGameUpdates(player, room_id_string)
			}
		}
	}
	l.clients[client.client_id] = client
	old_client.connection = nil
	old_client.deleted = true
	id_string := strconv.Itoa(int(client.client_id))
	client.send_message <- lobbyMessage{Sender: "server", Kind: "lobby-you-joined", Content: client.nickname, Data: id_string + "-reconnect"}
	l.broadcastMessage(lobbyMessage{Sender: "client-" + id_string, Kind: "lobby-joined", Content: client.nickname, Data: id_string + "-reconnect"})
}

func (l *Lobby) removeClient(client *Client) {
	if client.deleted {
		return // don't want to remove reconnected client
	}
	delete_client := time.NewTimer(1 * time.Hour)
	client.delete_timer = delete_client
	lobby_room := client.lobby_room
	go func() {
		<-delete_client.C
		client.deleted = true
		delete(l.clients, client.client_id)
	}()
	id_string := strconv.Itoa(int(client.client_id))
	if lobby_room != nil {
		lobby_room.removeClient(client, false)
	}
	l.broadcastMessage(lobbyMessage{Sender: "client-" + id_string, Kind: "lobby-left", Content: client.nickname, Data: id_string})
}

func (l *Lobby) createRoom(client *Client) {
	if client.lobby_room != nil {
		client.lobby_room.removeClient(client, false)
	}
	room_id := l.next_room_id
	l.next_room_id++
	room := CreateLobbyRoom(client, room_id, l)
	l.rooms[room.room_id] = room
	id_string := strconv.Itoa(int(room.room_id))
	client_id_string := strconv.Itoa(int(client.client_id))
	room_stringified, err := json.Marshal(room.ToFrontend())
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
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
	if room == nil || room.host == nil {
		return
	}
	room_id_string := strconv.Itoa(int(room.room_id))
	host_id_string := strconv.Itoa(int(room.host.client_id))
	l.broadcastMessage(lobbyMessage{
		Sender:  "room-" + room_id_string,
		Kind:    "room-renamed",
		Content: room.room_name,
		Data:    host_id_string,
	})
}

func (l *Lobby) updateRoomDescription(room *LobbyRoom) {
	if room == nil || room.host == nil {
		return
	}
	room_id_string := strconv.Itoa(int(room.room_id))
	host_id_string := strconv.Itoa(int(room.host.client_id))
	l.broadcastMessage(lobbyMessage{
		Sender:  "room-" + room_id_string,
		Kind:    "room-description-updated",
		Content: room.room_description,
		Data:    host_id_string,
	})
}

func (l *Lobby) removeRoom(room *LobbyRoom) {
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
	l.broadcastMessage(lobbyMessage{Sender: "server", Kind: "room-closed", Data: id_string})
}

func (l *Lobby) leaveRoom(data *ClientRoom) {
	data.room.removeClient(data.client, true)
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
		fmt.Printf("Broadcasting message {%s, %s, %s, %s}\n", message.Sender, message.Content, message.Data, message.Kind)
	}
	if util.Contains(client_to_lobby_messages, message.Kind) {
		send_id_string := strings.TrimPrefix(message.Sender, "client-")
		sender_id, err := strconv.ParseInt(send_id_string, 10, 0)
		if err != nil {
			sender_id = -1
		}
		for _, client := range l.clients {
			if client == nil || !client.valid() || client.client_id == uint64(sender_id) {
				continue
			}
			client.send_message <- message
		}
	} else if util.Contains(lobby_messages, message.Kind) {
		for _, client := range l.clients {
			if client == nil || !client.valid() {
				continue
			}
			client.send_message <- message
		}
	} else if util.Contains(client_to_room_messages, message.Kind) {
		send_split := strings.Split(message.Sender, "-")
		if len(send_split) < 3 {
			fmt.Fprintln(os.Stderr, "Sender not properly formed for a client to room message")
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
		fmt.Println("Sending chat from client", client_id)
		for _, client := range l.clients {
			if client == nil || !client.valid() || client.lobby_room == nil ||
				client.lobby_room.room_id != uint64(room_id) || client.client_id == uint64(client_id) {
				continue
			}
			client.send_message <- message
		}
	} else if util.Contains(room_messages, message.Kind) {
		room_id_string := strings.TrimPrefix(message.Sender, "room-")
		room_id, err := strconv.ParseInt(room_id_string, 10, 0)
		if err != nil {
			room_id = -1
		}
		for _, client := range l.clients {
			if client == nil || !client.valid() || client.lobby_room == nil || client.lobby_room.room_id != uint64(room_id) {
				continue
			}
			client.send_message <- message
		}
	} else {
		fmt.Fprintln(os.Stderr, "No logic for message kind")
	}
}

func (l *Lobby) setClientId(client *Client) {
	if client.client_id > 0 {
		fmt.Println("Client already has id:", client.client_id)
		return
	}
	client.client_id = l.next_client_id
	l.next_client_id++
}
