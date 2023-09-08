package lobby

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Lobby struct {
	next_room_id   uint64
	next_client_id uint64
	clients        map[uint64]*Client
	rooms          map[uint64]*LobbyRoom
	AddClient      chan *Client
	RemoveClient   chan *Client
	AddRoom        chan *websocket.Conn
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
		AddRoom:        make(chan *websocket.Conn),
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
		case client_connection := <-l.AddRoom:
			l.addRoom(client_connection)
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

func (l *Lobby) addClient(client *Client) {
	l.setClientId(client)
	l.clients[client.client_id] = client
	id_string := strconv.Itoa(int(client.client_id))
	client.send_message <- lobbyMessage{Sender: "server", Kind: "you-joined-lobby", Data: id_string}
}

func (l *Lobby) removeClient(client *Client) {
	delete(l.clients, client.client_id)
	client.close()
}

func (l *Lobby) addRoom(client_connection *websocket.Conn) {
	//
}

func (l *Lobby) removeRoom(room_id *uint64) {
	//
}

func (l *Lobby) broadcastMessage(message lobbyMessage) {
	fmt.Println(message.Sender, message.Content, message.Data, message.Kind)
}

func (l *Lobby) setClientId(client *Client) {
	if client.client_id > 0 {
		fmt.Println("Client already has id:", client.client_id)
		return
	}
	client.client_id = l.next_client_id
	l.next_client_id++
}
