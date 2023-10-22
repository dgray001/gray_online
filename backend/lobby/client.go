package lobby

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/dgray001/gray_online/game"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	read_wait            = 30 * time.Second
	read_limit           = 5120 // bytes
	ping_time            = 350 * time.Millisecond
	ping_broadcast_count = 4
	write_wait           = 10 * time.Second
)

type Client struct {
	client_id              uint64
	nickname               string
	connection             *websocket.Conn
	ping                   time.Duration
	ping_start             time.Time
	ping_broadcast_counter uint8
	send_message           chan lobbyMessage
	lobby                  *Lobby
	lobby_room             *LobbyRoom
	game                   game.Game
	delete_timer           *time.Timer
}

type lobbyMessage struct {
	Sender  string `json:"sender"`
	Kind    string `json:"kind"`
	Content string `json:"content"`
	Data    string `json:"data"`
}

func CreateClient(connection *websocket.Conn, nickname string, lobby *Lobby) *Client {
	client := Client{
		client_id:              0,
		nickname:               nickname,
		connection:             connection,
		ping:                   0,
		ping_start:             time.Now(),
		ping_broadcast_counter: 1,
		send_message:           make(chan lobbyMessage),
		lobby:                  lobby,
		lobby_room:             nil,
		game:                   nil,
	}
	go client.readMessages()
	go client.writeMessages()
	return &client
}

// Close send message channel
func (c *Client) close() {
	close(c.send_message)
	c.connection.Close()
}

func (c *Client) pingString() string {
	return strconv.FormatInt(int64(c.ping/time.Millisecond), 10)
}

func (c *Client) readMessages() {
	defer func() {
		c.lobby.RemoveClient <- c
		c.close()
	}()
	c.connection.SetReadLimit(read_limit)
	c.connection.SetReadDeadline(time.Now().Add(read_wait))
	c.connection.SetPongHandler(func(string) error {
		c.ping = time.Now().Sub(c.ping_start)
		c.connection.SetReadDeadline(time.Now().Add(read_wait))
		c.ping_broadcast_counter--
		ping_message := lobbyMessage{
			Sender:  "server",
			Kind:    "ping-update",
			Data:    strconv.FormatUint(c.client_id, 10),
			Content: c.pingString(),
		}
		if c.ping_broadcast_counter < 1 {
			c.ping_broadcast_counter = ping_broadcast_count
			c.lobby.broadcastMessage(ping_message)
		} else {
			c.send_message <- ping_message
		}
		return nil
	})
	for {
		var message lobbyMessage
		err := c.connection.ReadJSON(&message)
		if err != nil {
			fmt.Println("Error at message reader:", err)
			break
		}
		fmt.Printf("Receiving client message from %d: {%s, %s, %s, %s}\n",
			c.client_id, message.Sender, message.Content, message.Data, message.Kind)
		switch message.Kind {
		case "room-create":
			c.lobby.CreateRoom <- c
		case "room-join":
			room_id, err := strconv.Atoi(message.Data)
			if err != nil || room_id < 1 {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "room-join-failed", Content: "Invalid room id"}
				break
			}
			room := c.lobby.GetRoom(uint64(room_id))
			if room == nil {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "room-join-failed", Content: "Room doesn't exist"}
				break
			}
			room.JoinRoom <- MakeClientRoom(c, room)
		case "room-join-player":
			room_id, err := strconv.Atoi(message.Data)
			if err != nil || room_id < 1 {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "room-join-failed", Content: "Invalid room id"}
				break
			}
			room := c.lobby.GetRoom(uint64(room_id))
			if room == nil {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "room-join-failed", Content: "Room doesn't exist"}
				break
			}
			client_room := MakeClientRoom(c, room)
			client_room.bool_flag = true
			room.JoinRoom <- client_room
		case "room-join-viewer":
			room_id, err := strconv.Atoi(message.Data)
			if err != nil || room_id < 1 {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "room-join-failed", Content: "Invalid room id"}
				break
			}
			room := c.lobby.GetRoom(uint64(room_id))
			if room == nil {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "room-join-failed", Content: "Room doesn't exist"}
				break
			}
			client_room := MakeClientRoom(c, room)
			client_room.bool_flag = false
			room.JoinRoom <- client_room
		case "room-leave":
			room_id, err := strconv.Atoi(message.Data)
			if err != nil || room_id < 1 {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "room-leave-failed", Content: "Invalid room id"}
				break
			}
			room := c.lobby.GetRoom(uint64(room_id))
			if room == nil {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "room-leave-failed", Content: "Room doesn't exist"}
				break
			}
			if c.lobby_room == nil || c.lobby_room.room_id != room.room_id {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "room-leave-failed", Content: "Not in that room"}
				break
			}
			c.lobby.LeaveRoom <- MakeClientRoom(c, room)
		case "room-rename":
			room_id, err := strconv.Atoi(message.Data)
			if err != nil || room_id < 1 {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "room-rename-failed", Content: "Invalid room id"}
				break
			}
			room := c.lobby.GetRoom(uint64(room_id))
			if room == nil {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "room-rename-failed", Content: "Room doesn't exist"}
				break
			}
			if room.host.client_id != c.client_id {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "room-rename-failed", Content: "Not room host"}
				break
			}
			room.room_name = message.Content
			c.lobby.RenameRoom <- room
		case "room-kick":
			room_id, err := strconv.Atoi(message.Data)
			if err != nil || room_id < 1 {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "room-kick-failed", Content: "Invalid room id"}
				break
			}
			room := c.lobby.GetRoom(uint64(room_id))
			if room == nil {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "room-kick-failed", Content: "Room doesn't exist"}
				break
			}
			if room.host.client_id != c.client_id {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "room-kick-failed", Content: "Not room host"}
				break
			}
			client_id, err := strconv.Atoi(message.Content)
			if err != nil || client_id < 1 {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "room-kick-failed", Content: "Invalid client id"}
				break
			}
			client := c.lobby.GetClient(uint64(client_id))
			if client == nil {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "room-kick-failed", Content: "Client doesn't exist"}
				break
			}
			c.lobby.KickClientFromRoom <- MakeClientRoom(client, room)
		case "room-promote":
			room_id, err := strconv.Atoi(message.Data)
			if err != nil || room_id < 1 {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "room-promote-failed", Content: "Invalid room id"}
				break
			}
			room := c.lobby.GetRoom(uint64(room_id))
			if room == nil {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "room-promote-failed", Content: "Room doesn't exist"}
				break
			}
			if room.host.client_id != c.client_id {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "room-promote-failed", Content: "Not room host"}
				break
			}
			client_id, err := strconv.Atoi(message.Content)
			if err != nil || client_id < 1 {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "room-promote-failed", Content: "Invalid client id"}
				break
			}
			client := c.lobby.GetClient(uint64(client_id))
			if client == nil {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "room-promote-failed", Content: "Client doesn't exist"}
				break
			}
			c.lobby.PromotePlayerInRoom <- MakeClientRoom(client, room)
		case "room-set-viewer":
			room_id, err := strconv.Atoi(message.Data)
			if err != nil || room_id < 1 {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "room-set-viewer-failed", Content: "Invalid room id"}
				break
			}
			room := c.lobby.GetRoom(uint64(room_id))
			if room == nil {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "room-set-viewer-failed", Content: "Room doesn't exist"}
				break
			}
			client_id, err := strconv.Atoi(message.Content)
			if err != nil || client_id < 1 {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "room-set-viewer-failed", Content: "Invalid client id"}
				break
			}
			if room.host.client_id != c.client_id && client_id != int(c.client_id) {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "room-set-viewer-failed", Content: "Not room host"}
				break
			}
			client := c.lobby.GetClient(uint64(client_id))
			if client == nil {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "room-set-viewer-failed", Content: "Client doesn't exist"}
				break
			}
			c.lobby.RoomSetViewer <- MakeClientRoom(client, room)
		case "room-set-player":
			room_id, err := strconv.Atoi(message.Data)
			if err != nil || room_id < 1 {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "room-set-player-failed", Content: "Invalid room id"}
				break
			}
			room := c.lobby.GetRoom(uint64(room_id))
			if room == nil {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "room-set-player-failed", Content: "Room doesn't exist"}
				break
			}
			client_id, err := strconv.Atoi(message.Content)
			if err != nil || client_id < 1 {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "room-set-player-failed", Content: "Invalid client id"}
				break
			}
			if room.host.client_id != c.client_id && client_id != int(c.client_id) {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "room-set-player-failed", Content: "Not room host"}
				break
			}
			client := c.lobby.GetClient(uint64(client_id))
			if client == nil {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "room-set-player-failed", Content: "Client doesn't exist"}
				break
			}
			c.lobby.RoomSetPlayer <- MakeClientRoom(client, room)
		case "room-settings-update":
			room_id, err := strconv.Atoi(message.Data)
			if err != nil || room_id < 1 {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "room-settings-update-failed", Content: "Invalid room id"}
				break
			}
			room := c.lobby.GetRoom(uint64(room_id))
			if room == nil {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "room-settings-update-failed", Content: "Room doesn't exist"}
				break
			}
			if room.host.client_id != c.client_id {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "room-settings-update-failed", Content: "Not room host"}
				break
			}
			settings := GameSettings{}
			unmarshal_err := json.Unmarshal([]byte(message.Content), &settings)
			if unmarshal_err != nil {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "room-settings-update-failed", Content: "Data in an improper context"}
				break
			}
			c.lobby.UpdateSettings <- MakeSettingsRoom(&settings, room)
		case "room-launch":
			room_id, err := strconv.Atoi(message.Data)
			if err != nil || room_id < 1 {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "room-launch-failed", Content: "Invalid room id"}
				break
			}
			room := c.lobby.GetRoom(uint64(room_id))
			if room == nil {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "room-launch-failed", Content: "Room doesn't exist"}
				break
			}
			if room.host.client_id != c.client_id {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "room-launch-failed", Content: "Not room host"}
				break
			}
			if !room.launchable() {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "room-launch-failed", Content: "Room not launchable"}
				break
			}
			c.lobby.LaunchGame <- room
		case "room-refresh":
			room_id, err := strconv.Atoi(message.Data)
			if err != nil || room_id < 1 {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "room-refresh-failed", Content: "Invalid room id"}
				break
			}
			room := c.lobby.GetRoom(uint64(room_id))
			if room == nil {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "room-refresh-failed", Content: "Room doesn't exist"}
				break
			}
			room_stringified, err := json.Marshal(room.ToFrontend())
			if err != nil {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "room-refresh-failed", Content: "Room not stringified properly"}
				break
			}
			c.send_message <- lobbyMessage{Sender: "room-" + message.Data, Kind: "refresh-room", Content: string(room_stringified), Data: message.Data}
		case "game-connected":
			game_id, err := strconv.Atoi(message.Data)
			if err != nil || game_id < 1 {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "game-connected-failed", Content: "Invalid game id"}
				break
			}
			if c.lobby_room == nil {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "game-connected-failed", Content: "Client not in lobby"}
				break
			}
			if c.lobby_room.game == nil {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "game-connected-failed", Content: "Lobby not launched"}
				break
			}
			if game_id != int(c.lobby_room.game.GetId()) {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "game-connected-failed", Content: "Incorrect game id"}
				break
			}
			c.lobby_room.PlayerConnected <- c
		case "game-chat":
			if c.lobby_room == nil {
				break
			}
			c.lobby_room.broadcast <- message
		case "game-update":
			if c.lobby_room == nil || c.lobby_room.game == nil {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "game-update-failed", Content: "Not in game"}
				break
			}
			action := gin.H{}
			err := json.Unmarshal([]byte(message.Content), &action)
			if err != nil {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "game-update-failed", Content: "Couldn't parse action"}
				break
			}
			player_action := game.PlayerAction{Kind: message.Data, Client_id: int(c.client_id), Action: action}
			c.lobby_room.PlayerAction <- player_action
		case "game-get-update":
			if c.lobby_room == nil || c.lobby_room.game == nil {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "game-get-update-failed", Content: "Not in game"}
				break
			}
			update_id, err := strconv.Atoi(message.Data)
			if err != nil {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "game-get-update-failed", Content: "Invalid update id"}
				break
			}
			c.lobby_room.game.GetBase().ResendPlayerUpdate(c.client_id, update_id)
		case "game-resend-last-update":
			if c.lobby_room == nil || c.lobby_room.game == nil {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "game-resend-last-update-failed", Content: "Not in game"}
				break
			}
			c.lobby_room.game.GetBase().ResendLastUpdate(c.client_id)
		case "game-resend-waiting-room":
			if c.lobby_room == nil || c.lobby_room.game == nil {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "game-resend-waiting-room-failed", Content: "Not in game"}
				break
			}
			game_base := c.lobby_room.game.GetBase()
			if game_base == nil {
				c.send_message <- lobbyMessage{Sender: "server", Kind: "game-resend-waiting-room-failed", Content: "No game base returned"}
				break
			}
			room_id_string := strconv.Itoa(int(c.lobby_room.room_id))
			if game_base.GameStarted() {
				c.send_message <- lobbyMessage{Sender: "room-" + room_id_string, Kind: "game-start"}
			}
		case "lobby-chat":
			fallthrough
		case "room-chat":
			if c.lobby == nil {
				break
			}
			c.lobby.broadcast <- message
		default:
			fmt.Println("Message kind unknown: " + message.Kind)
		}
	}
}

func (c *Client) writeMessages() {
	ticker := time.NewTicker(ping_time)
	defer func() {
		c.connection.Close()
		ticker.Stop()
	}()
	for {
		select {
		case message, ok := <-c.send_message:
			c.connection.SetWriteDeadline(time.Now().Add(write_wait))
			if !ok {
				c.connection.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			err := c.connection.WriteJSON(message)
			if err != nil {
				fmt.Println("Error at message writer:", err)
				break
			}
		case <-ticker.C:
			c.connection.SetWriteDeadline(time.Now().Add(write_wait))
			err := c.connection.WriteMessage(websocket.PingMessage, nil)
			if err != nil {
				fmt.Println("Error at ping writer:", err)
				break
			}
			c.ping_start = time.Now()
		}
	}
}

func (c *Client) valid() bool {
	if c == nil {
		return false
	}
	if c.client_id < 1 {
		return false
	}
	if c.connection == nil {
		return false
	}
	if c.delete_timer != nil {
		return false
	}
	return true
}

func (c *Client) ToFrontend() gin.H {
	client := gin.H{
		"client_id": strconv.Itoa(int(c.client_id)),
		"nickname":  c.nickname,
		"ping":      c.pingString(),
	}
	if c.lobby_room != nil {
		client["room_id"] = c.lobby_room.room_id
	}
	return client
}
