package lobby

import (
	"encoding/json"
	"fmt"
	"os"
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
	deleted                bool
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
		send_message:           make(chan lobbyMessage, 4),
		lobby:                  lobby,
		lobby_room:             nil,
		game:                   nil,
		deleted:                false,
	}
	go client.readMessages()
	go client.writeMessages()
	return &client
}

func (c *Client) GetNickname() string {
	return c.nickname
}

func (c *Client) close() {
	if c == nil {
		return
	}
	if c.connection != nil {
		c.connection.Close()
	}
}

func (c *Client) send(lm lobbyMessage) {
	if c.send_message == nil {
		return
	}
	select {
	case c.send_message <- lm:
	default:
	}
}

func (c *Client) pingStringLocked() string {
	return strconv.FormatInt(int64(c.ping/time.Millisecond), 10)
}

func (c *Client) readMessages() {
	// only ever becomes nil, never a different connection
	connection := c.connection
	defer func() {
		c.lobby.RemoveClient <- c
		connection.Close()
	}()
	connection.SetReadLimit(read_limit)
	connection.SetReadDeadline(time.Now().Add(read_wait))
	connection.SetPongHandler(func(string) error {
		c.lobby.mu.Lock()
		if !c.validDebugLocked(false) {
			c.lobby.mu.Unlock()
			return nil
		}
		c.ping = time.Since(c.ping_start)
		c.ping_broadcast_counter--
		broadcast_wide := c.ping_broadcast_counter < 1
		if broadcast_wide {
			c.ping_broadcast_counter = ping_broadcast_count
		}
		ping_message := lobbyMessage{
			Sender:  "server",
			Kind:    "ping-update",
			Data:    strconv.FormatUint(c.client_id, 10),
			Content: c.pingStringLocked(),
		}
		c.lobby.mu.Unlock()
		connection.SetReadDeadline(time.Now().Add(read_wait))
		if broadcast_wide {
			c.lobby.broadcastMessage(ping_message)
		} else {
			c.send(ping_message)
		}
		return nil
	})
	for {
		c.lobby.mu.Lock()
		deleted := c.deleted
		c.lobby.mu.Unlock()
		if deleted {
			break
		}
		var message lobbyMessage
		err := connection.ReadJSON(&message)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error at message reader: "+err.Error())
			break
		}
		fmt.Printf("Receiving client message from %d: {%s, %s, %s, %s}\n",
			c.client_id, message.Sender, message.Content, message.Data, message.Kind)
		c.handleMessage(message)
	}
}

func (c *Client) handleMessage(message lobbyMessage) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintln(os.Stderr, "recovered from panic handling message from", c.client_id, ":", r)
		}
	}()
	switch message.Kind {
	case "room-create":
		c.lobby.CreateRoom <- c
	case "room-join":
		room_id, err := strconv.Atoi(message.Data)
		if err != nil || room_id < 1 {
			c.send(lobbyMessage{Sender: "server", Kind: "room-join-failed", Content: "Invalid room id"})
			break
		}
		room := c.lobby.GetRoom(uint64(room_id))
		if room == nil {
			c.send(lobbyMessage{Sender: "server", Kind: "room-join-failed", Content: "Room doesn't exist"})
			break
		}
		client_room := MakeClientRoom(c, room)
		client_room.bool_flag = true // default to join as player
		room.JoinRoom <- client_room
	case "room-join-player":
		room_id, err := strconv.Atoi(message.Data)
		if err != nil || room_id < 1 {
			c.send(lobbyMessage{Sender: "server", Kind: "room-join-failed", Content: "Invalid room id"})
			break
		}
		room := c.lobby.GetRoom(uint64(room_id))
		if room == nil {
			c.send(lobbyMessage{Sender: "server", Kind: "room-join-failed", Content: "Room doesn't exist"})
			break
		}
		client_room := MakeClientRoom(c, room)
		client_room.bool_flag = true
		room.JoinRoom <- client_room
	case "room-join-viewer":
		room_id, err := strconv.Atoi(message.Data)
		if err != nil || room_id < 1 {
			c.send(lobbyMessage{Sender: "server", Kind: "room-join-failed", Content: "Invalid room id"})
			break
		}
		room := c.lobby.GetRoom(uint64(room_id))
		if room == nil {
			c.send(lobbyMessage{Sender: "server", Kind: "room-join-failed", Content: "Room doesn't exist"})
			break
		}
		client_room := MakeClientRoom(c, room)
		client_room.bool_flag = false
		room.JoinRoom <- client_room
	case "room-leave":
		room_id, err := strconv.Atoi(message.Data)
		if err != nil || room_id < 1 {
			c.send(lobbyMessage{Sender: "server", Kind: "room-leave-failed", Content: "Invalid room id"})
			break
		}
		room := c.lobby.GetRoom(uint64(room_id))
		if room == nil {
			c.send(lobbyMessage{Sender: "server", Kind: "room-leave-failed", Content: "Room doesn't exist"})
			break
		}
		lobby_room := c.getLobbyRoom()
		if lobby_room == nil || lobby_room.room_id != room.room_id {
			fmt.Println(lobby_room, room)
			c.send(lobbyMessage{Sender: "server", Kind: "room-leave-failed", Content: "Not in that room"})
			break
		}
		c.lobby.LeaveRoom <- MakeClientRoom(c, room)
	case "room-rename":
		room_id, err := strconv.Atoi(message.Data)
		if err != nil || room_id < 1 {
			c.send(lobbyMessage{Sender: "server", Kind: "room-rename-failed", Content: "Invalid room id"})
			break
		}
		room := c.lobby.GetRoom(uint64(room_id))
		if room == nil {
			c.send(lobbyMessage{Sender: "server", Kind: "room-rename-failed", Content: "Room doesn't exist"})
			break
		}
		if room.getHostId() != c.client_id {
			c.send(lobbyMessage{Sender: "server", Kind: "room-rename-failed", Content: "Not room host"})
			break
		}
		if room.gameStarted() {
			c.send(lobbyMessage{Sender: "server", Kind: "room-rename-failed", Content: "Game started"})
			break
		}
		c.lobby.mu.Lock()
		room.room_name = message.Content
		c.lobby.mu.Unlock()
		c.lobby.RenameRoom <- room
	case "room-update-description":
		room_id, err := strconv.Atoi(message.Data)
		if err != nil || room_id < 1 {
			c.send(lobbyMessage{Sender: "server", Kind: "room-update-description-failed", Content: "Invalid room id"})
			break
		}
		room := c.lobby.GetRoom(uint64(room_id))
		if room == nil {
			c.send(lobbyMessage{Sender: "server", Kind: "room-update-description-failed", Content: "Room doesn't exist"})
			break
		}
		if room.getHostId() != c.client_id {
			c.send(lobbyMessage{Sender: "server", Kind: "room-update-description-failed", Content: "Not room host"})
			break
		}
		if room.gameStarted() {
			c.send(lobbyMessage{Sender: "server", Kind: "room-update-description-failed", Content: "Game started"})
			break
		}
		c.lobby.mu.Lock()
		room.room_description = message.Content
		c.lobby.mu.Unlock()
		c.lobby.UpdateRoomDescription <- room
	case "room-kick":
		room_id, err := strconv.Atoi(message.Data)
		if err != nil || room_id < 1 {
			c.send(lobbyMessage{Sender: "server", Kind: "room-kick-failed", Content: "Invalid room id"})
			break
		}
		room := c.lobby.GetRoom(uint64(room_id))
		if room == nil {
			c.send(lobbyMessage{Sender: "server", Kind: "room-kick-failed", Content: "Room doesn't exist"})
			break
		}
		if room.getHostId() != c.client_id {
			c.send(lobbyMessage{Sender: "server", Kind: "room-kick-failed", Content: "Not room host"})
			break
		}
		if room.gameStarted() {
			c.send(lobbyMessage{Sender: "server", Kind: "room-kick-failed", Content: "Game started"})
			break
		}
		client_id, err := strconv.Atoi(message.Content)
		if err != nil || client_id < 1 {
			c.send(lobbyMessage{Sender: "server", Kind: "room-kick-failed", Content: "Invalid client id"})
			break
		}
		client := c.lobby.GetClient(uint64(client_id))
		if client == nil {
			c.send(lobbyMessage{Sender: "server", Kind: "room-kick-failed", Content: "Client doesn't exist"})
			break
		}
		c.lobby.KickClientFromRoom <- MakeClientRoom(client, room)
	case "room-promote":
		room_id, err := strconv.Atoi(message.Data)
		if err != nil || room_id < 1 {
			c.send(lobbyMessage{Sender: "server", Kind: "room-promote-failed", Content: "Invalid room id"})
			break
		}
		room := c.lobby.GetRoom(uint64(room_id))
		if room == nil {
			c.send(lobbyMessage{Sender: "server", Kind: "room-promote-failed", Content: "Room doesn't exist"})
			break
		}
		if room.getHostId() != c.client_id {
			c.send(lobbyMessage{Sender: "server", Kind: "room-promote-failed", Content: "Not room host"})
			break
		}
		if room.gameStarted() {
			c.send(lobbyMessage{Sender: "server", Kind: "room-promote-failed", Content: "Game started"})
			break
		}
		client_id, err := strconv.Atoi(message.Content)
		if err != nil || client_id < 1 {
			c.send(lobbyMessage{Sender: "server", Kind: "room-promote-failed", Content: "Invalid client id"})
			break
		}
		client := c.lobby.GetClient(uint64(client_id))
		if client == nil {
			c.send(lobbyMessage{Sender: "server", Kind: "room-promote-failed", Content: "Client doesn't exist"})
			break
		}
		c.lobby.PromotePlayerInRoom <- MakeClientRoom(client, room)
	case "room-set-viewer":
		room_id, err := strconv.Atoi(message.Data)
		if err != nil || room_id < 1 {
			c.send(lobbyMessage{Sender: "server", Kind: "room-set-viewer-failed", Content: "Invalid room id"})
			break
		}
		room := c.lobby.GetRoom(uint64(room_id))
		if room == nil {
			c.send(lobbyMessage{Sender: "server", Kind: "room-set-viewer-failed", Content: "Room doesn't exist"})
			break
		}
		client_id, err := strconv.Atoi(message.Content)
		if err != nil || client_id < 1 {
			c.send(lobbyMessage{Sender: "server", Kind: "room-set-viewer-failed", Content: "Invalid client id"})
			break
		}
		if room.getHostId() != c.client_id && client_id != int(c.client_id) {
			c.send(lobbyMessage{Sender: "server", Kind: "room-set-viewer-failed", Content: "Not room host"})
			break
		}
		if room.gameStarted() {
			c.send(lobbyMessage{Sender: "server", Kind: "room-set-viewer-failed", Content: "Game started"})
			break
		}
		client := c.lobby.GetClient(uint64(client_id))
		if client == nil {
			c.send(lobbyMessage{Sender: "server", Kind: "room-set-viewer-failed", Content: "Client doesn't exist"})
			break
		}
		c.lobby.RoomSetViewer <- MakeClientRoom(client, room)
	case "room-set-player":
		room_id, err := strconv.Atoi(message.Data)
		if err != nil || room_id < 1 {
			c.send(lobbyMessage{Sender: "server", Kind: "room-set-player-failed", Content: "Invalid room id"})
			break
		}
		room := c.lobby.GetRoom(uint64(room_id))
		if room == nil {
			c.send(lobbyMessage{Sender: "server", Kind: "room-set-player-failed", Content: "Room doesn't exist"})
			break
		}
		client_id, err := strconv.Atoi(message.Content)
		if err != nil || client_id < 1 {
			c.send(lobbyMessage{Sender: "server", Kind: "room-set-player-failed", Content: "Invalid client id"})
			break
		}
		if room.getHostId() != c.client_id && client_id != int(c.client_id) {
			c.send(lobbyMessage{Sender: "server", Kind: "room-set-player-failed", Content: "Not room host"})
			break
		}
		if room.gameStarted() {
			c.send(lobbyMessage{Sender: "server", Kind: "room-set-player-failed", Content: "Game started"})
			break
		}
		client := c.lobby.GetClient(uint64(client_id))
		if client == nil {
			c.send(lobbyMessage{Sender: "server", Kind: "room-set-player-failed", Content: "Client doesn't exist"})
			break
		}
		c.lobby.RoomSetPlayer <- MakeClientRoom(client, room)
	case "room-settings-update":
		room_id, err := strconv.Atoi(message.Data)
		if err != nil || room_id < 1 {
			c.send(lobbyMessage{Sender: "server", Kind: "room-settings-update-failed", Content: "Invalid room id"})
			break
		}
		room := c.lobby.GetRoom(uint64(room_id))
		if room == nil {
			c.send(lobbyMessage{Sender: "server", Kind: "room-settings-update-failed", Content: "Room doesn't exist"})
			break
		}
		if room.getHostId() != c.client_id {
			c.send(lobbyMessage{Sender: "server", Kind: "room-settings-update-failed", Content: "Not room host"})
			break
		}
		if room.gameStarted() {
			c.send(lobbyMessage{Sender: "server", Kind: "room-settings-update-failed", Content: "Game started"})
			break
		}
		settings := GameSettings{}
		unmarshal_err := json.Unmarshal([]byte(message.Content), &settings)
		if unmarshal_err != nil {
			c.send(lobbyMessage{Sender: "server", Kind: "room-settings-update-failed", Content: "Data in an improper context"})
			break
		}
		room.UpdateSettings <- &settings
	case "room-launch":
		room_id, err := strconv.Atoi(message.Data)
		if err != nil || room_id < 1 {
			c.send(lobbyMessage{Sender: "server", Kind: "room-launch-failed", Content: "Invalid room id"})
			break
		}
		room := c.lobby.GetRoom(uint64(room_id))
		if room == nil {
			c.send(lobbyMessage{Sender: "server", Kind: "room-launch-failed", Content: "Room doesn't exist"})
			break
		}
		if room.getHostId() != c.client_id {
			c.send(lobbyMessage{Sender: "server", Kind: "room-launch-failed", Content: "Not room host"})
			break
		}
		launchable, launchable_error := room.launchable()
		if !launchable {
			c.send(lobbyMessage{Sender: "server", Kind: "room-launch-failed", Content: "Room not launchable: " + launchable_error})
			break
		}
		c.lobby.LaunchGame <- room
	case "room-refresh":
		room_id, err := strconv.Atoi(message.Data)
		if err != nil || room_id < 1 {
			c.send(lobbyMessage{Sender: "server", Kind: "room-refresh-failed", Content: "Invalid room id"})
			break
		}
		room := c.lobby.GetRoom(uint64(room_id))
		if room == nil {
			c.send(lobbyMessage{Sender: "server", Kind: "room-refresh-failed", Content: "Room doesn't exist"})
			break
		}
		room_stringified, err := json.Marshal(room.ToFrontend())
		if err != nil {
			c.send(lobbyMessage{Sender: "server", Kind: "room-refresh-failed", Content: "Room not stringified properly"})
			break
		}
		c.send(lobbyMessage{Sender: "room-" + message.Data, Kind: "room-refreshed", Content: string(room_stringified), Data: message.Data})
	case "game-connected":
		game_id, err := strconv.Atoi(message.Data)
		if err != nil || game_id < 1 {
			c.send(lobbyMessage{Sender: "server", Kind: "game-connected-failed", Content: "Invalid game id"})
			break
		}
		lobby_room := c.getLobbyRoom()
		if lobby_room == nil {
			c.send(lobbyMessage{Sender: "server", Kind: "game-connected-failed", Content: "Client not in lobby room"})
			break
		}
		room_game := lobby_room.getGame()
		if room_game == nil {
			c.send(lobbyMessage{Sender: "server", Kind: "game-connected-failed", Content: "Lobby not launched"})
			break
		}
		if game_id != int(game.Game_GetId(room_game)) {
			c.send(lobbyMessage{Sender: "server", Kind: "game-connected-failed", Content: "Incorrect game id"})
			break
		}
		lobby_room.PlayerConnected <- c
	case "game-chat":
		lobby_room := c.getLobbyRoom()
		if lobby_room == nil {
			break
		}
		expected_sender := "game-" + strconv.FormatUint(c.client_id, 10)
		if message.Sender != expected_sender {
			fmt.Fprintln(os.Stderr, "Client", c.client_id, "sent game-chat with mismatched sender, dropping:", message.Sender)
			break
		}
		lobby_room.broadcast <- message
	case "game-update":
		lobby_room := c.getLobbyRoom()
		if lobby_room == nil || lobby_room.getGame() == nil {
			c.send(lobbyMessage{Sender: "server", Kind: "game-update-failed", Content: "Not in game"})
			break
		}
		action := gin.H{}
		if message.Content != "" {
			err := json.Unmarshal([]byte(message.Content), &action)
			if err != nil {
				fmt.Println(err)
				c.send(lobbyMessage{Sender: "server", Kind: "game-update-failed", Content: "Couldn't parse action"})
				break
			}
		}
		player_action := game.PlayerAction{Kind: message.Data, Client_id: int(c.client_id), Action: action}
		lobby_room.PlayerAction <- player_action
	case "game-exit":
		lobby_room := c.getLobbyRoom()
		if lobby_room == nil || lobby_room.getGame() == nil {
			if client_game := c.getGame(); client_game != nil && client_game.GetBase().GameEnded() {
				break
			}
			c.send(lobbyMessage{Sender: "server", Kind: "game-update-failed", Content: "Not in game"})
			break
		}
		lobby_room.PlayerDisconnected <- c
	case "game-get-update":
		lobby_room := c.getLobbyRoom()
		if lobby_room == nil {
			if client_game := c.getGame(); client_game != nil && client_game.GetBase().GameEnded() {
				break
			}
			c.send(lobbyMessage{Sender: "server", Kind: "game-get-update-failed", Content: "Not in game"})
			break
		}
		room_game := lobby_room.getGame()
		if room_game == nil {
			if client_game := c.getGame(); client_game != nil && client_game.GetBase().GameEnded() {
				break
			}
			c.send(lobbyMessage{Sender: "server", Kind: "game-get-update-failed", Content: "Not in game"})
			break
		}
		update_id, err := strconv.Atoi(message.Data)
		if err != nil {
			c.send(lobbyMessage{Sender: "server", Kind: "game-get-update-failed", Content: "Invalid update id"})
			break
		}
		room_game.GetBase().ResendPlayerUpdate(c.client_id, update_id)
	case "game-resend-last-update":
		lobby_room := c.getLobbyRoom()
		if lobby_room == nil {
			if client_game := c.getGame(); client_game != nil && client_game.GetBase().GameEnded() {
				break
			}
			c.send(lobbyMessage{Sender: "server", Kind: "game-resend-last-update-failed", Content: "Not in game"})
			break
		}
		room_game := lobby_room.getGame()
		if room_game == nil {
			if client_game := c.getGame(); client_game != nil && client_game.GetBase().GameEnded() {
				break
			}
			c.send(lobbyMessage{Sender: "server", Kind: "game-resend-last-update-failed", Content: "Not in game"})
			break
		}
		room_game.GetBase().ResendLastUpdate(c.client_id)
	case "game-resend-waiting-room":
		lobby_room := c.getLobbyRoom()
		if lobby_room == nil || lobby_room.getGame() == nil {
			c.send(lobbyMessage{Sender: "server", Kind: "game-resend-waiting-room-failed", Content: "Not in game"})
			break
		}
		game_base := lobby_room.getGame().GetBase()
		if game_base == nil {
			c.send(lobbyMessage{Sender: "server", Kind: "game-resend-waiting-room-failed", Content: "No game base returned"})
			break
		}
		room_id_string := strconv.Itoa(int(lobby_room.room_id))
		if game_base.GameStarted() {
			c.send(lobbyMessage{Sender: "room-" + room_id_string, Kind: "game-start"})
		} else {
			c.send(lobbyMessage{Sender: "room-" + room_id_string, Kind: "game-resend-waiting-room-ack"})
		}
	case "lobby-chat":
		fallthrough
	case "room-chat":
		if c.lobby == nil {
			break
		}
		if message.Kind == "lobby-chat" {
			expected_sender := "client-" + strconv.FormatUint(c.client_id, 10)
			if message.Sender != expected_sender {
				fmt.Fprintln(os.Stderr, "Client", c.client_id, "sent lobby-chat with mismatched sender, dropping:", message.Sender)
				break
			}
		} else {
			lobby_room := c.getLobbyRoom()
			if lobby_room == nil {
				break
			}
			expected_sender := "room-" + strconv.Itoa(int(lobby_room.room_id)) + "-" + strconv.FormatUint(c.client_id, 10)
			if message.Sender != expected_sender {
				fmt.Fprintln(os.Stderr, "Client", c.client_id, "sent room-chat with mismatched sender, dropping:", message.Sender)
				break
			}
		}
		c.lobby.broadcast <- message
	default:
		fmt.Fprintln(os.Stderr, "Message kind unknown: "+message.Kind)
	}
}

func (c *Client) writeMessages() {
	// only ever becomes nil, never a different connection
	connection := c.connection
	ticker := time.NewTicker(ping_time)
	defer func() {
		connection.Close()
		ticker.Stop()
	}()
	for {
		c.lobby.mu.Lock()
		deleted := c.deleted
		c.lobby.mu.Unlock()
		if deleted {
			break
		}
		c.writeMessagesOnce(connection, ticker)
	}
}

func (c *Client) writeMessagesOnce(connection *websocket.Conn, ticker *time.Ticker) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintln(os.Stderr, "recovered from panic in writeMessages for client", c.client_id, ":", r)
		}
	}()
	select {
	case message := <-c.send_message:
		connection.SetWriteDeadline(time.Now().Add(write_wait))
		err := connection.WriteJSON(message)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error at client", c.client_id, "message writer: ", err.Error())
			c.lobby.mu.Lock()
			c.deleted = true
			c.lobby.mu.Unlock()
		}
	case <-ticker.C:
		connection.SetWriteDeadline(time.Now().Add(write_wait))
		err := connection.WriteMessage(websocket.PingMessage, nil)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error at client", c.client_id, "ping writer: ", err.Error())
			c.lobby.mu.Lock()
			c.deleted = true
			c.lobby.mu.Unlock()
			break
		}
		c.lobby.mu.Lock()
		c.ping_start = time.Now()
		c.lobby.mu.Unlock()
	}
}

func (c *Client) validDebugLocked(debug bool) bool {
	if c == nil {
		if debug {
			fmt.Fprintln(os.Stderr, "Client invalid because null", c)
		}
		return false
	}
	if c.client_id < 1 {
		if debug {
			fmt.Fprintln(os.Stderr, "Client invalid because id < 1:", c)
		}
		return false
	}
	if c.connection == nil {
		if debug {
			fmt.Fprintln(os.Stderr, "Client invalid because connection nil", c)
		}
		return false
	}
	if c.delete_timer != nil {
		if debug {
			fmt.Fprintln(os.Stderr, "Client invalid because delete timer not nil", c)
		}
		return false
	}
	if c.deleted {
		if debug {
			fmt.Fprintln(os.Stderr, "Client invalid because deleted", c)
		}
		return false
	}
	return true
}

func (c *Client) validDebug(debug bool) bool {
	if c == nil {
		return false
	}
	c.lobby.mu.Lock()
	defer c.lobby.mu.Unlock()
	return c.validDebugLocked(debug)
}

func (c *Client) validLocked() bool {
	return c.validDebugLocked(false)
}

func (c *Client) valid() bool {
	return c.validDebug(false)
}

func (c *Client) gameNil() bool {
	return c.game == nil || c.game.GetBase() == nil
}

func (c *Client) getLobbyRoom() *LobbyRoom {
	c.lobby.mu.Lock()
	defer c.lobby.mu.Unlock()
	return c.lobby_room
}

func (c *Client) getGame() game.Game {
	c.lobby.mu.Lock()
	defer c.lobby.mu.Unlock()
	return c.game
}

func (c *Client) toFrontendLocked() gin.H {
	client := gin.H{
		"client_id": strconv.Itoa(int(c.client_id)),
		"nickname":  c.nickname,
		"ping":      c.pingStringLocked(),
	}
	if c.lobby_room != nil {
		client["room_id"] = c.lobby_room.room_id
	}
	return client
}

func (c *Client) ToFrontend() gin.H {
	c.lobby.mu.Lock()
	defer c.lobby.mu.Unlock()
	return c.toFrontendLocked()
}
