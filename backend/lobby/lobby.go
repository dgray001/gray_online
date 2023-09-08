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
	RemoveRoom     chan *uint64
	broadcast      chan lobbyMessage
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
		RemoveRoom:     make(chan *uint64),
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
		case room_id := <-l.RemoveRoom:
			l.removeRoom(room_id)
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
	l.broadcastMessage(lobbyMessage{Sender: "client-" + id_string, Kind: "lobby-left", Content: client.nickname, Data: id_string})
}

func (l *Lobby) createRoom(client *Client) {
	room_id := l.next_room_id
	l.next_room_id++
	room := CreateLobbyRoom(client, room_id)
	l.rooms[room.room_id] = room
	id_string := strconv.Itoa(int(room.room_id))
	client_id_string := strconv.Itoa(int(client.client_id))
	l.broadcastMessage(lobbyMessage{Sender: "server", Kind: "room-created", Content: client_id_string, Data: id_string})
}

func (l *Lobby) removeRoom(room_id *uint64) {
	//
}

var (
	client_to_lobby_messages = []string{"lobby-join", "lobby-left", "lobby-chat"}
	to_all_messages          = []string{"room-created"}
)

func (l *Lobby) broadcastMessage(message lobbyMessage) {
	fmt.Printf("Broadcasting message {%s, %s, %s, %s}\n", message.Sender, message.Content, message.Data, message.Kind)
	if util.Contains(client_to_lobby_messages, message.Kind) {
		send_id_string := strings.TrimPrefix(message.Sender, "client-")
		sender_id, err := strconv.ParseInt(send_id_string, 10, 0)
		if err != nil {
			sender_id = 0
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
