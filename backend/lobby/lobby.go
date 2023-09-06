package lobby

import (
	"fmt"

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

func (l *Lobby) addRoom(client_connection *websocket.Conn) {
	fmt.Println("new connection", client_connection.LocalAddr().String())
}

func (l *Lobby) removeRoom(room_id *uint64) {
	//
}
