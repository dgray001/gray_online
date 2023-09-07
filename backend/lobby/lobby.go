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
	AddRoom        chan *websocket.Conn
	RemoveRoom     chan *uint64
}

func CreateLobby() *Lobby {
	return &Lobby{
		next_room_id:   1,
		next_client_id: 1,
		clients:        make(map[uint64]*Client),
		rooms:          make(map[uint64]*LobbyRoom),
		AddClient:      make(chan *Client),
		AddRoom:        make(chan *websocket.Conn),
		RemoveRoom:     make(chan *uint64),
	}
}

func (l *Lobby) Run() {
	for {
		select {
		case client := <-l.AddClient:
			l.addClient(client)
		case client_connection := <-l.AddRoom:
			l.addRoom(client_connection)
		case room_id := <-l.RemoveRoom:
			l.removeRoom(room_id)
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
	client.send_message <- lobbyMessage{Sender: "server", Kind: "join-lobby", Data: strconv.Itoa(int(client.client_id))}
}

func (l *Lobby) addRoom(client_connection *websocket.Conn) {
	//
}

func (l *Lobby) removeRoom(room_id *uint64) {
	//
}

func (l *Lobby) setClientId(client *Client) {
	if client.client_id > 0 {
		fmt.Println("Client already has id:", client.client_id)
		return
	}
	client.client_id = l.next_client_id
	l.next_client_id++
}
