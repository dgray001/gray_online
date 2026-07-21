package lobby

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/euchre"
	"github.com/dgray001/gray_online/game/games/fiddlesticks"
	"github.com/dgray001/gray_online/game/games/risq"
	"github.com/dgray001/gray_online/game/games/test_game"
	"github.com/dgray001/gray_online/util"
	"github.com/gin-gonic/gin"
)

type GameStateRequest struct {
	ClientId uint64
	IsViewer bool
	Reply    chan gin.H
}

type LobbyRoom struct {
	room_id           uint64
	room_name         string
	room_description  string
	host              *Client
	players           map[uint64]*Client
	viewers           map[uint64]*Client
	lobby             *Lobby
	game              game.Game
	game_settings     *GameSettings
	delete_timer      *time.Timer
	ended_game_id     uint64
	game_delete_timer *time.Timer
	// lobby room and game channels
	broadcast          chan lobbyMessage
	JoinRoom           chan *ClientRoom
	LeaveRoom          chan *ClientRoom
	PlayerReplaced     chan *Client
	UpdateSettings     chan *GameSettings
	PlayerConnected    chan *Client
	PlayerDisconnected chan *Client
	PlayerAction       chan game.PlayerAction
	GameEnded          chan string
	GameStateRequest   chan *GameStateRequest
}

func CreateLobbyRoom(host *Client, room_id uint64, lobby *Lobby) *LobbyRoom {
	room := LobbyRoom{
		room_id:          room_id,
		room_name:        fmt.Sprintf("%s's room", host.nickname),
		room_description: "",
		host:             host,
		players:          make(map[uint64]*Client),
		viewers:          make(map[uint64]*Client),
		lobby:            lobby,
		game_settings: &GameSettings{
			MaxPlayers: 8,
			MaxViewers: 16,
			GameType:   0,
		},
		broadcast:          make(chan lobbyMessage, 4),
		JoinRoom:           make(chan *ClientRoom, 4),
		LeaveRoom:          make(chan *ClientRoom, 4),
		PlayerReplaced:     make(chan *Client, 4),
		UpdateSettings:     make(chan *GameSettings, 4),
		PlayerConnected:    make(chan *Client, 4),
		PlayerDisconnected: make(chan *Client, 4),
		PlayerAction:       make(chan game.PlayerAction, 12),
		GameEnded:          make(chan string, 4),
		GameStateRequest:   make(chan *GameStateRequest, 4),
	}
	room.players[host.client_id] = host
	host.lobby_room = &room
	go room.run()
	return &room
}

func (r *LobbyRoom) run() {
	for {
		r.runOnce()
	}
}

func (r *LobbyRoom) runOnce() {
	defer func() {
		if rec := recover(); rec != nil {
			fmt.Fprintln(os.Stderr, "recovered from panic in room", r.room_id, "actor loop:", rec)
		}
	}()
	select {
	case message := <-r.broadcast:
		r.broadcastMessage(message)
	case data := <-r.JoinRoom:
		r.addClient(data.client, data.bool_flag)
	case data := <-r.LeaveRoom:
		r.removeClient(data.client, data.bool_flag)
	case client := <-r.PlayerReplaced:
		r.replaceClient(client)
	case client := <-r.PlayerDisconnected:
		r.playerDisconnected(client)
	case settings := <-r.UpdateSettings:
		r.updateSettings(settings)
	case client := <-r.PlayerConnected:
		r.lobby.mu.Lock()
		client.game = r.game
		r.lobby.mu.Unlock()
		start_game := r.game.GetBase().PlayerConnected(client.client_id)
		client_id_string := strconv.Itoa(int(client.client_id))
		room_id_string := strconv.Itoa(int(r.room_id))
		r.broadcastMessage(lobbyMessage{Sender: "room-" + room_id_string, Kind: "game-player-connected", Data: client_id_string})
		if start_game {
			for client_id, player := range r.players {
				go player.playerGameUpdates(r.game.GetBase().Players[client_id], room_id_string)
			}
			for client_id, viewer := range r.viewers {
				go viewer.viewerGameUpdates(r.game.GetBase().Viewers[client_id], room_id_string)
			}
			go r.gameBaseUpdates(r.game.GetBase())
			r.broadcastMessage(lobbyMessage{Sender: "room-" + room_id_string, Kind: "game-start", Data: ""})
			game.Game_StartGame(r.game)
		}
	case action := <-r.PlayerAction:
		fmt.Println("Received player action: ", action)
		if !r.gameNil() {
			r.game.PlayerAction(r.game.GetBase().AddAction(action))
		}
	case message := <-r.GameEnded:
		room_id_string := strconv.Itoa(int(r.room_id))
		r.lobby.broadcastMessage(lobbyMessage{Sender: "room-" + room_id_string, Kind: "room-game-over", Data: message})
		ended_game_id := game.Game_GetId(r.game)
		r.lobby.mu.Lock()
		r.game = nil
		r.ended_game_id = ended_game_id
		r.game_delete_timer = time.NewTimer(10 * time.Minute)
		r.lobby.mu.Unlock()
		go func() {
			<-r.game_delete_timer.C
			r.evictEndedGame(ended_game_id)
		}()
	case req := <-r.GameStateRequest:
		if r.game == nil {
			req.Reply <- gin.H{}
		} else {
			req.Reply <- r.game.ToFrontend(req.ClientId, req.IsViewer)
		}
	}
}

func (c *Client) playerGameUpdates(player *game.Player, room_id_string string) {
	if player == nil {
		return
	}
	fmt.Println("Begin player game updates for", c.client_id, player.Player_id)
	select {
	case <-player.FlushConnections:
	default:
	}
updates_loop:
	for {
		if c.deleted {
			break
		}
		fmt.Println("Iterate player game updates for", c.client_id, player.Player_id, c.valid())
		select {
		case message := <-player.Updates:
			fmt.Println("Received player game update", message.Id, "for", c.client_id, player.Player_id)
			encoded_message, err := json.Marshal(message.Content)
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				break
			}
			message_id_string := strconv.Itoa(message.Id)
			if c.validDebug(true) {
				c.send_message <- lobbyMessage{Sender: "room-" + room_id_string + "-" + message_id_string,
					Kind: "game-update", Data: message.Kind, Content: string(encoded_message)}
			} else {
				fmt.Fprintln(os.Stderr, "Room failed to send update message to player", c.client_id)
			}
		case message := <-player.FailedUpdates:
			encoded_message, err := json.Marshal(message.Content)
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				break
			}
			message_id_string := strconv.Itoa(message.Id)
			if c.validDebug(true) {
				c.send_message <- lobbyMessage{Sender: "room-" + room_id_string + "-" + message_id_string,
					Kind: "game-failed-update", Data: message.Kind, Content: string(encoded_message)}
			} else {
				fmt.Fprintln(os.Stderr, "Room failed to send failed update message to player", c.client_id)
			}
		case game_over := <-player.FlushConnections:
			if game_over {
				fmt.Println("Player game updates flushed because game is over")
			} else {
				fmt.Println("Player game updates flushed because player disconnected")
			}
			break updates_loop
		}
	}
	fmt.Println("End player game updates for", c.client_id, player.Player_id)
}

func (c *Client) viewerGameUpdates(viewer *game.Viewer, room_id_string string) {
	for {
		if c.deleted || viewer == nil {
			break
		}
		select {
		case message := <-viewer.Updates:
			encoded_message, err := json.Marshal(message.Content)
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				break
			}
			message_id_string := strconv.Itoa(message.Id)
			if c.validDebug(true) {
				c.send_message <- lobbyMessage{Sender: "room-" + room_id_string + "-" + message_id_string,
					Kind: "game-update", Data: message.Kind, Content: string(encoded_message)}
			} else {
				fmt.Fprintln(os.Stderr, "Room failed to send update message to viewer", c.client_id)
			}
		}
	}
}

func (r *LobbyRoom) gameBaseUpdates(game_base *game.GameBase) {
	for {
		if game_base == nil {
			break
		}
		select {
		case <-game_base.ViewerUpdates:
		case message := <-game_base.GameEndedChannel:
			r.GameEnded <- message
			return
		}
	}
}

func (r *LobbyRoom) replaceClient(client *Client) {
	r.lobby.mu.Lock()
	if r.host.client_id == client.client_id {
		r.host = client
	}
	if util.MapContains(r.players, client.client_id) {
		r.players[client.client_id] = client
	}
	if util.MapContains(r.viewers, client.client_id) {
		r.viewers[client.client_id] = client
	}
	game_started := r.gameStartedLocked()
	current_game := r.game
	if r.delete_timer != nil {
		r.delete_timer.Stop()
		r.delete_timer = nil
	}
	r.lobby.mu.Unlock()
	if game_started {
		game.Game_PlayerReconnected(current_game, client.client_id)
	}
}

func (r *LobbyRoom) broadcastMessage(message lobbyMessage) {
	if message.Kind != "ping-update" {
		fmt.Printf("Broadcasting lobby (%d) message {%s, %s, %s, %s}\n", r.room_id, message.Sender, message.Content, message.Data, message.Kind)
	}
	r.lobby.mu.Lock()
	defer r.lobby.mu.Unlock()
	for client_id, client := range r.players {
		if client == nil || !client.validDebugLocked(true) {
			fmt.Fprintln(os.Stderr, "Room failed to send message to player", client_id)
			continue
		}
		client.send(message)
	}
	for client_id, client := range r.viewers {
		if client == nil || !client.validDebugLocked(true) {
			fmt.Fprintln(os.Stderr, "Room failed to send message to viewer", client_id)
			continue
		}
		client.send(message)
	}
}

func (r *LobbyRoom) gameNilLocked() bool {
	return r.game == nil || r.game.GetBase() == nil
}

func (r *LobbyRoom) gameNil() bool {
	r.lobby.mu.Lock()
	defer r.lobby.mu.Unlock()
	return r.gameNilLocked()
}

func (r *LobbyRoom) getGame() game.Game {
	r.lobby.mu.Lock()
	defer r.lobby.mu.Unlock()
	return r.game
}

func (r *LobbyRoom) getHostId() uint64 {
	r.lobby.mu.Lock()
	defer r.lobby.mu.Unlock()
	return r.host.client_id
}

func (r *LobbyRoom) gameStartedLocked() bool {
	return r.game != nil && r.game.GetBase() != nil && r.game.GetBase().GameStarted() && !r.game.GetBase().GameEnded()
}

func (r *LobbyRoom) gameStarted() bool {
	r.lobby.mu.Lock()
	defer r.lobby.mu.Unlock()
	return r.gameStartedLocked()
}

func (r *LobbyRoom) addClient(c *Client, join_as_player bool) {
	r.lobby.mu.Lock()
	game_started := r.gameStartedLocked()
	current_game := r.game
	r.lobby.mu.Unlock()
	if game_started {
		if game.Game_PlayerReconnected(current_game, c.client_id) {
			r.lobby.mu.Lock()
			c.lobby_room = r
			r.players[c.client_id] = c
			if r.host != nil && r.host.client_id == c.client_id {
				r.host = c
			}
			r.lobby.mu.Unlock()
			client_id_string := strconv.Itoa(int(c.client_id))
			room_id_string := strconv.Itoa(int(r.room_id))
			r.lobby.broadcastMessage(lobbyMessage{Sender: "room-" + room_id_string, Kind: "room-joined-player", Data: client_id_string})
			return
		}
		join_as_player = false
	}
	r.lobby.mu.Lock()
	old_lobby_room := c.lobby_room
	r.lobby.mu.Unlock()
	if old_lobby_room != nil {
		if old_lobby_room.room_id == r.room_id {
			c.send(lobbyMessage{Sender: "server", Kind: "room-join-failed", Content: "Already in room"})
			return
		}
		old_lobby_room.LeaveRoom <- &ClientRoom{client: c, room: old_lobby_room, bool_flag: true}
	}
	r.lobby.mu.Lock()
	if join_as_player && !r.spaceForPlayer() {
		join_as_player = false
	}
	if !join_as_player && !r.spaceForViewer() {
		r.lobby.mu.Unlock()
		c.send(lobbyMessage{Sender: "server", Kind: "room-join-failed", Content: "No space in room"})
		return
	}
	if join_as_player {
		r.players[c.client_id] = c
		if r.host.client_id == c.client_id {
			r.host = c
		}
	} else {
		r.viewers[c.client_id] = c
	}
	c.lobby_room = r
	message_kind := "room-joined-player"
	var new_viewer_game *game.GameBase
	if !join_as_player {
		message_kind = "room-joined-viewer"
		if !r.gameNilLocked() {
			new_viewer_game = r.game.GetBase()
		}
	}
	r.lobby.mu.Unlock()
	if new_viewer_game != nil {
		new_viewer_game.Viewers[c.client_id] = game.CreateViewer(c.client_id, c.nickname)
	}
	client_id_string := strconv.Itoa(int(c.client_id))
	room_id_string := strconv.Itoa(int(r.room_id))
	r.lobby.broadcastMessage(lobbyMessage{Sender: "room-" + room_id_string, Kind: message_kind, Data: client_id_string})
}

func (r *LobbyRoom) spaceForPlayer() bool {
	return r.game_settings != nil && int(r.game_settings.MaxPlayers) > len(r.players)
}

func (r *LobbyRoom) spaceForViewer() bool {
	return r.game_settings != nil && int(r.game_settings.MaxViewers) > len(r.viewers)
}

func (r *LobbyRoom) removeClient(c *Client, client_leaves bool) {
	r.lobby.mu.Lock()
	game_started := r.gameStartedLocked()
	is_host_leaving_unstarted := r.host != nil && r.host.client_id == c.client_id && !game_started
	r.lobby.mu.Unlock()
	if is_host_leaving_unstarted {
		// TODO: make someone else the room host if game not started
		fmt.Println("Removing room since host left unstarted game")
		r.lobby.removeRoom(r)
		return
	}
	if game_started {
		r.playerDisconnected(c)
	}
	r.lobby.mu.Lock()
	if !game_started || client_leaves {
		delete(r.players, c.client_id)
		if c.lobby_room != nil && c.lobby_room.room_id == r.room_id {
			c.lobby_room = nil
		}
	}
	if util.MapContains(r.viewers, c.client_id) {
		delete(r.viewers, c.client_id)
		if c.lobby_room != nil && c.lobby_room.room_id == r.room_id {
			c.lobby_room = nil
		}
	}
	r.lobby.mu.Unlock()
	r.checkEndedGameEmpty()
	client_id_string := strconv.Itoa(int(c.client_id))
	room_id_string := strconv.Itoa(int(r.room_id))
	r.lobby.broadcastMessage(lobbyMessage{Sender: "room-" + room_id_string, Kind: "room-left", Data: client_id_string})
}

func (r *LobbyRoom) evictEndedGame(game_id uint64) {
	r.lobby.mu.Lock()
	defer r.lobby.mu.Unlock()
	if r.ended_game_id != game_id {
		return
	}
	delete(r.lobby.games, game_id)
	r.ended_game_id = 0
	r.game_delete_timer = nil
}

// evicts the pending ended game early if no player or viewer is still connected
func (r *LobbyRoom) checkEndedGameEmpty() {
	r.lobby.mu.Lock()
	game_id := r.ended_game_id
	if game_id == 0 {
		r.lobby.mu.Unlock()
		return
	}
	for _, player := range r.players {
		if player.validDebugLocked(false) {
			r.lobby.mu.Unlock()
			return
		}
	}
	for _, viewer := range r.viewers {
		if viewer.validDebugLocked(false) {
			r.lobby.mu.Unlock()
			return
		}
	}
	if r.game_delete_timer != nil {
		r.game_delete_timer.Stop()
	}
	r.lobby.mu.Unlock()
	r.evictEndedGame(game_id)
}

/** This needs to be the way we disconnect players from a game */
func (r *LobbyRoom) playerDisconnected(c *Client) {
	r.lobby.mu.Lock()
	current_game := r.game
	r.lobby.mu.Unlock()
	if game.Game_PlayerDisconnected(current_game, c.client_id) {
		delete_room := time.NewTimer(45 * time.Second)
		r.lobby.mu.Lock()
		r.delete_timer = delete_room
		r.lobby.mu.Unlock()
		go func() {
			<-delete_room.C
			fmt.Println("Removing room since everyone left for 45 seconds")
			r.lobby.removeRoom(r)
		}()
	}
	client_id_string := strconv.Itoa(int(c.client_id))
	room_id_string := strconv.Itoa(int(r.room_id))
	r.broadcastMessage(lobbyMessage{Sender: "room-" + room_id_string, Kind: "game-player-disconnected", Data: client_id_string})
}

func (r *LobbyRoom) kickClient(c *Client) {
	r.lobby.mu.Lock()
	if r.host.client_id == c.client_id {
		r.lobby.mu.Unlock()
		return
	}
	if c.lobby_room != nil && c.lobby_room.room_id == r.room_id {
		c.lobby_room = nil
	}
	delete(r.players, c.client_id)
	delete(r.viewers, c.client_id)
	r.lobby.mu.Unlock()
	client_id_string := strconv.Itoa(int(c.client_id))
	room_id_string := strconv.Itoa(int(r.room_id))
	r.lobby.broadcastMessage(lobbyMessage{Sender: "room-" + room_id_string, Kind: "room-kicked", Data: client_id_string})
}

func (r *LobbyRoom) promotePlayer(c *Client) {
	r.lobby.mu.Lock()
	if r.host.client_id == c.client_id {
		r.lobby.mu.Unlock()
		return
	}
	if c.lobby_room == nil || c.lobby_room.room_id != r.room_id {
		r.lobby.mu.Unlock()
		return
	}
	r.host = c
	r.lobby.mu.Unlock()
	client_id_string := strconv.Itoa(int(c.client_id))
	room_id_string := strconv.Itoa(int(r.room_id))
	r.lobby.broadcastMessage(lobbyMessage{Sender: "room-" + room_id_string, Kind: "room-promoted", Data: client_id_string})
}

func (r *LobbyRoom) setViewer(c *Client) {
	r.lobby.mu.Lock()
	if r.host.client_id == c.client_id || r.players[c.client_id] == nil {
		r.lobby.mu.Unlock()
		return
	}
	delete(r.players, c.client_id)
	r.viewers[c.client_id] = c
	r.lobby.mu.Unlock()
	client_id_string := strconv.Itoa(int(c.client_id))
	room_id_string := strconv.Itoa(int(r.room_id))
	r.lobby.broadcastMessage(lobbyMessage{Sender: "room-" + room_id_string, Kind: "room-viewer-set", Data: client_id_string})
}

func (r *LobbyRoom) setPlayer(c *Client) {
	r.lobby.mu.Lock()
	if r.host.client_id == c.client_id || r.viewers[c.client_id] == nil {
		r.lobby.mu.Unlock()
		return
	}
	delete(r.viewers, c.client_id)
	r.players[c.client_id] = c
	r.lobby.mu.Unlock()
	client_id_string := strconv.Itoa(int(c.client_id))
	room_id_string := strconv.Itoa(int(r.room_id))
	r.lobby.broadcastMessage(lobbyMessage{Sender: "room-" + room_id_string, Kind: "room-player-set", Data: client_id_string})
}

func (r *LobbyRoom) updateSettings(s *GameSettings) {
	settings_stringified, err := json.Marshal(s.ToFrontend())
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}
	r.lobby.mu.Lock()
	r.game_settings = s
	r.lobby.mu.Unlock()
	room_id_string := strconv.Itoa(int(r.room_id))
	r.lobby.broadcastMessage(lobbyMessage{
		Sender: "room-" + room_id_string,
		Kind:   "room-settings-updated",
		Data:   string(settings_stringified),
	})
}

func (r *LobbyRoom) launchGame(game_id uint64) (game.Game, error) {
	r.lobby.mu.Lock()
	launchable, launchable_error := r.launchableLocked()
	if !launchable {
		r.lobby.mu.Unlock()
		message := fmt.Sprintf("Cannot launch game of id %d: %s", game_id, launchable_error)
		fmt.Fprintln(os.Stderr, message)
		return nil, errors.New(message)
	}
	base_game := game.CreateBaseGame(game_id, r.game_settings.GameType, r.game_settings.GameSpecificSettings)
	base_game.RequestToFrontend = r.RequestToFrontend
	for _, player := range r.players {
		game.CreatePlayer(player.client_id, player.nickname, base_game)
	}
	for _, viewer := range r.viewers {
		base_game.Viewers[viewer.client_id] = game.CreateViewer(viewer.client_id, viewer.nickname)
	}
	game_type := r.game_settings.GameType
	r.lobby.mu.Unlock()

	var err error = nil
	var new_game game.Game = nil
	switch game_type {
	case game.GameType_FIDDLESTICKS:
		new_game, err = fiddlesticks.CreateGame(base_game, r.PlayerAction)
	case game.GameType_EUCHRE:
		new_game, err = euchre.CreateGame(base_game)
	case game.GameType_RISQ:
		new_game, err = risq.CreateGame(base_game)
	case game.GameType_TEST_GAME:
		new_game, err = test_game.CreateGame(base_game)
	default:
		err = fmt.Errorf("%s", fmt.Sprintf("GameType not recognized: %d", game_type))
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return nil, err
	}
	r.lobby.mu.Lock()
	r.game = new_game
	r.lobby.mu.Unlock()
	room_id_string := strconv.Itoa(int(r.room_id))
	game_id_string := strconv.Itoa(int(game_id))
	r.lobby.broadcastMessage(lobbyMessage{Sender: "room-" + room_id_string, Kind: "room-launched", Data: game_id_string})
	return new_game, nil
}

func (r *LobbyRoom) GetGame() game.Game {
	return r.game
}

func (r *LobbyRoom) RequestToFrontend(client_id uint64, is_viewer bool) (gin.H, error) {
	reply := make(chan gin.H, 1)
	r.GameStateRequest <- &GameStateRequest{ClientId: client_id, IsViewer: is_viewer, Reply: reply}
	select {
	case result := <-reply:
		return result, nil
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("room %d timed out replying to game state request", r.room_id)
	}
}

func (r *LobbyRoom) launchableLocked() (bool, string) {
	if !r.validLocked() {
		return false, "Room invalid"
	}
	if r.game_settings == nil {
		return false, "No game settings"
	}
	settings_launchable, settings_launchable_error := r.game_settings.Launchable()
	if !settings_launchable {
		return false, settings_launchable_error
	}
	if !r.gameNilLocked() {
		return false, "Game already launched"
	}
	return true, ""
}

func (r *LobbyRoom) launchable() (bool, string) {
	r.lobby.mu.Lock()
	defer r.lobby.mu.Unlock()
	return r.launchableLocked()
}

func (r *LobbyRoom) validLocked() bool {
	if r.room_id < 1 {
		return false
	}
	if !r.gameNilLocked() && !r.game.Valid() {
		return false
	}
	if r.gameNilLocked() && (r.host == nil || !r.host.validDebugLocked(true)) {
		return false
	}
	if len(r.players) > int(r.game_settings.MaxPlayers) {
		return false
	}
	if len(r.viewers) > int(r.game_settings.MaxViewers) {
		return false
	}
	// TODO: keep track of who is DC'ed and validate from that
	/*for _, client := range r.players {
	    if client == nil || !client.valid() {
	      return false
	    }
	  }
	  for _, client := range r.viewers {
	    if client == nil || !client.valid() {
	      return false
	    }
	  }*/
	return true
}

func (r *LobbyRoom) valid() bool {
	r.lobby.mu.Lock()
	defer r.lobby.mu.Unlock()
	return r.validLocked()
}

func (r *LobbyRoom) toFrontendLocked() gin.H {
	room := gin.H{
		"room_id":          strconv.Itoa(int(r.room_id)),
		"room_name":        r.room_name,
		"room_description": r.room_description,
		"game_settings":    r.game_settings.ToFrontend(),
	}
	if r.host != nil {
		room["host"] = r.host.toFrontendLocked()
	}
	players := []gin.H{}
	for _, player := range r.players {
		if player != nil {
			players = append(players, player.toFrontendLocked())
		}
	}
	room["players"] = players
	viewers := []gin.H{}
	for _, viewer := range r.viewers {
		if viewer != nil {
			viewers = append(viewers, viewer.toFrontendLocked())
		}
	}
	room["viewers"] = viewers
	if !r.gameNilLocked() {
		room["game_id"] = strconv.Itoa(int(game.Game_GetId(r.game)))
	}
	return room
}

func (r *LobbyRoom) ToFrontend() gin.H {
	r.lobby.mu.Lock()
	defer r.lobby.mu.Unlock()
	return r.toFrontendLocked()
}
