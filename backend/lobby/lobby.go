package lobby

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/dgray001/gray_online/util"
	"github.com/gin-gonic/gin"
)

type Lobby struct {
	next_room_id   uint64
	next_client_id uint64
	clients        map[uint64]*Client
	rooms          map[uint64]*LobbyRoom
	AddClient      chan *Client
	RemoveClient   chan *Client
	CreateRoom     chan *Client
	RenameRoom     chan *LobbyRoom
	RemoveRoom     chan *LobbyRoom
	JoinRoom       chan *ClientRoom
	LeaveRoom      chan *ClientRoom
	broadcast      chan lobbyMessage
}

type ClientRoom struct {
	client *Client
	room   *LobbyRoom
}

func MakeClientRoom(client *Client, room *LobbyRoom) *ClientRoom {
	return &ClientRoom{
		client: client,
		room:   room,
	}
}

func CreateLobby() *Lobby {
	return &Lobby{
		next_room_id:   1,
		next_client_id: 1,
		clients:        make(map[uint64]*Client),
		rooms:          make(map[uint64]*LobbyRoom),
		AddClient:      make(chan *Client),
		RemoveClient:   make(chan *Client),
		CreateRoom:     make(chan *Client),
		RenameRoom:     make(chan *LobbyRoom),
		RemoveRoom:     make(chan *LobbyRoom),
		JoinRoom:       make(chan *ClientRoom),
		LeaveRoom:      make(chan *ClientRoom),
		broadcast:      make(chan lobbyMessage),
	}
}

func (l *Lobby) Run() {
	for {
		select {
		case client := <-l.AddClient:
			l.addClient(client)
		case client := <-l.RemoveClient:
			l.removeClient(client)
		case client := <-l.CreateRoom:
			l.createRoom(client)
		case room := <-l.RenameRoom:
			l.renameRoom(room)
		case room := <-l.RemoveRoom:
			l.removeRoom(room)
		case data := <-l.JoinRoom:
			l.joinRoom(data)
		case data := <-l.LeaveRoom:
			l.leaveRoom(data)
		case message := <-l.broadcast:
			l.broadcastMessage(message)
		}
	}
}

func (l *Lobby) GetRooms() []gin.H {
	rooms := []gin.H{}
	for _, room := range l.rooms {
		if room.valid() {
			rooms = append(rooms, room.toFrontend())
		}
	}
	return rooms
}

func (l *Lobby) GetUsers() []gin.H {
	users := []gin.H{}
	for _, user := range l.clients {
		if user.valid() {
			users = append(users, user.toFrontend())
		}
	}
	return users
}

func (l *Lobby) GetClient(client_id uint64) *Client {
	return l.clients[client_id]
}

func (l *Lobby) GetRoom(room_id uint64) *LobbyRoom {
	return l.rooms[room_id]
}

func (l *Lobby) addClient(client *Client) {
	l.setClientId(client)
	l.clients[client.client_id] = client
	id_string := strconv.Itoa(int(client.client_id))
	client.send_message <- lobbyMessage{Sender: "server", Kind: "lobby-you-joined", Content: client.nickname, Data: id_string}
}

func (l *Lobby) removeClient(client *Client) {
	delete(l.clients, client.client_id)
	client.close()
	id_string := strconv.Itoa(int(client.client_id))
	if client.lobby_room != nil && client.lobby_room.host != nil && client.lobby_room.host.client_id == client.client_id {
		l.removeRoom(client.lobby_room)
	}
	l.broadcastMessage(lobbyMessage{Sender: "client-" + id_string, Kind: "lobby-left", Content: client.nickname, Data: id_string})
}

func (l *Lobby) createRoom(client *Client) {
	if client.lobby_room != nil {
		client.lobby_room.removeClient(client)
	}
	room_id := l.next_room_id
	l.next_room_id++
	room := CreateLobbyRoom(client, room_id, l)
	l.rooms[room.room_id] = room
	id_string := strconv.Itoa(int(room.room_id))
	client_id_string := strconv.Itoa(int(client.client_id))
	l.broadcastMessage(lobbyMessage{Sender: "server", Kind: "room-created", Content: client_id_string, Data: id_string})
}

func (l *Lobby) renameRoom(room *LobbyRoom) {
	if room == nil || room.host == nil {
		return
	}
	room_id_string := strconv.Itoa(int(room.room_id))
	host_id_string := strconv.Itoa(int(room.host.client_id))
	l.broadcastMessage(lobbyMessage{
		Sender:  "room-" + room_id_string,
		Kind:    "room-rename",
		Content: room.room_name,
		Data:    host_id_string,
	})
}

func (l *Lobby) removeRoom(room *LobbyRoom) {
	delete(l.rooms, room.room_id)
	id_string := strconv.Itoa(int(room.room_id))
	if room.host != nil {
		room.host.lobby_room = nil
	}
	for _, joinee := range room.clients {
		if joinee != nil {
			joinee.lobby_room = nil
		}
	}
	l.broadcastMessage(lobbyMessage{Sender: "room-" + id_string, Kind: "room-closed", Data: id_string})
}

func (l *Lobby) joinRoom(data *ClientRoom) {
	if data.client.lobby_room != nil {
		data.client.lobby_room.removeClient(data.client)
	}
	data.room.addClient(data.client)
}

func (l *Lobby) leaveRoom(data *ClientRoom) {
	data.room.removeClient(data.client)
}

var (
	client_to_lobby_messages = []string{"lobby-join", "lobby-left", "lobby-chat"}
	to_all_messages          = []string{"room-created", "room-closed", "room-join", "room-leave"}
	client_to_room_messages  = []string{"room-chat"}
	host_to_room_messages    = []string{"room-rename"}
)

func (l *Lobby) broadcastMessage(message lobbyMessage) {
	fmt.Printf("Broadcasting message {%s, %s, %s, %s}\n", message.Sender, message.Content, message.Data, message.Kind)
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
			select {
			case client.send_message <- message:
			default:
				l.removeClient(client)
			}
		}
	} else if util.Contains(to_all_messages, message.Kind) {
		for _, client := range l.clients {
			if client == nil || !client.valid() {
				continue
			}
			select {
			case client.send_message <- message:
			default:
				l.removeClient(client)
			}
		}
	} else if util.Contains(client_to_room_messages, message.Kind) {
		send_split := strings.Split(message.Sender, "-")
		if len(send_split) < 3 {
			fmt.Println("Sender not properly formed for a client to room message")
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
		for _, client := range l.clients {
			if client == nil || !client.valid() || client.lobby_room == nil ||
				client.lobby_room.room_id != uint64(room_id) || client.client_id == uint64(client_id) {
				continue
			}
			select {
			case client.send_message <- message:
			default:
				l.removeClient(client)
			}
		}
	} else if util.Contains(host_to_room_messages, message.Kind) {
		room_id_string := strings.TrimPrefix(message.Sender, "room-")
		room_id, err := strconv.ParseInt(room_id_string, 10, 0)
		if err != nil {
			room_id = -1
		}
		host_id, err := strconv.ParseInt(message.Data, 10, 0)
		if err != nil {
			host_id = -1
		}
		for _, client := range l.clients {
			if client == nil || !client.valid() || client.client_id == uint64(host_id) ||
				client.lobby_room == nil || client.lobby_room.room_id == uint64(room_id) {
				continue
			}
			select {
			case client.send_message <- message:
			default:
				l.removeClient(client)
			}
		}
	} else {
		fmt.Println("No logic for message kind")
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
