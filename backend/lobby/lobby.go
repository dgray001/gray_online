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
	rooms          map[uint64]*LobbyRoom
	AddRoom        chan *websocket.Conn
	RemoveRoom     chan *uint64
}

func CreateLobby() *Lobby {
	return &Lobby{
		next_room_id:   1,
		next_client_id: 1,
		rooms:          make(map[uint64]*LobbyRoom),
		AddRoom:        make(chan *websocket.Conn),
		RemoveRoom:     make(chan *uint64),
	}
}

func (l *Lobby) Run() {
	for {
		select {
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

func (l *Lobby) addRoom(client_connection *websocket.Conn) {
	client := createClient(client_connection)
	client.send_message <- "client created"
	l.setClientId(client)
	room_id := l.next_room_id
	l.next_room_id++
	room := createRoom(client, room_id)
	l.rooms[room_id] = room
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
	client.send_message <- "set id: " + strconv.Itoa(int(l.next_client_id))
	l.next_client_id++
}
