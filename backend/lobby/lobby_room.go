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

type LobbyRoom struct {
	room_id          uint64
	room_name        string
	room_description string
	host             *Client
	players          map[uint64]*Client
	viewers          map[uint64]*Client
	lobby            *Lobby
	game             game.Game
	game_settings    *GameSettings
	delete_timer     *time.Timer
	// lobby room and game channels
	// TODO: move lobby room channels from lobby to lobby room
	broadcast       chan lobbyMessage
	JoinRoom        chan *ClientRoom
	UpdateSettings  chan *GameSettings
	PlayerConnected chan *Client
	PlayerAction    chan game.PlayerAction
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
		broadcast:       make(chan lobbyMessage, 4),
		JoinRoom:        make(chan *ClientRoom, 4),
		UpdateSettings:  make(chan *GameSettings, 4),
		PlayerConnected: make(chan *Client, 4),
		PlayerAction:    make(chan game.PlayerAction, 12),
	}
	room.players[host.client_id] = host
	host.lobby_room = &room
	go room.run()
	return &room
}

func (r *LobbyRoom) run() {
	for {
		select {
		case message := <-r.broadcast:
			r.broadcastMessage(message)
		case data := <-r.JoinRoom:
			r.addClient(data.client, data.bool_flag)
		case settings := <-r.UpdateSettings:
			r.updateSettings(settings)
		case client := <-r.PlayerConnected:
			client.game = r.game
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
				go r.gameBaseUpdates(r.game.GetBase(), room_id_string)
				r.broadcastMessage(lobbyMessage{Sender: "room-" + room_id_string, Kind: "game-start", Data: ""})
				game.Game_StartGame(r.game)
			}
		case action := <-r.PlayerAction:
			fmt.Println("Received player action: ", action)
			if !r.gameNil() {
				r.game.PlayerAction(r.game.GetBase().AddAction(action))
			}
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
			break
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

func (r *LobbyRoom) gameBaseUpdates(game_base *game.GameBase, room_id_string string) {
	for {
		if game_base == nil {
			break
		}
		select {
		case <-game_base.ViewerUpdates:
		case message := <-game_base.GameEndedChannel:
			r.lobby.broadcastMessage(lobbyMessage{Sender: "room-" + room_id_string, Kind: "room-game-over", Data: message})
			r.game = nil
		}
	}
}

func (r *LobbyRoom) replaceClient(client *Client) {
	if r.host.client_id == client.client_id {
		r.host = client
	}
	if util.MapContains(r.players, client.client_id) {
		r.players[client.client_id] = client
	}
	if util.MapContains(r.viewers, client.client_id) {
		r.viewers[client.client_id] = client
	}
	if r.gameStarted() {
		game.Game_PlayerReconnected(r.game, client.client_id)
	}
	if r.delete_timer != nil {
		r.delete_timer.Stop()
		r.delete_timer = nil
	}
}

func (r *LobbyRoom) broadcastMessage(message lobbyMessage) {
	if message.Kind != "ping-update" {
		fmt.Printf("Broadcasting lobby (%d) message {%s, %s, %s, %s}\n", r.room_id, message.Sender, message.Content, message.Data, message.Kind)
	}
	for _, client := range r.players {
		if client == nil || !client.validDebug(true) {
			fmt.Fprintln(os.Stderr, "Room failed to send message to player", client.client_id)
			continue
		}
		client.send(message)
	}
	for _, client := range r.viewers {
		if client == nil || !client.validDebug(true) {
			fmt.Fprintln(os.Stderr, "Room failed to send message to viewer", client.client_id)
			continue
		}
		client.send(message)
	}
}

func (r *LobbyRoom) gameNil() bool {
	return r.game == nil || r.game.GetBase() == nil
}

func (r *LobbyRoom) gameStarted() bool {
	return r.game != nil && r.game.GetBase() != nil && r.game.GetBase().GameStarted() && !r.game.GetBase().GameEnded()
}

func (r *LobbyRoom) addClient(c *Client, join_as_player bool) {
	if r.gameStarted() {
		if game.Game_PlayerReconnected(r.game, c.client_id) {
			c.lobby_room = r
			r.players[c.client_id] = c
			if r.host != nil && r.host.client_id == c.client_id {
				r.host = c
			}
			client_id_string := strconv.Itoa(int(c.client_id))
			room_id_string := strconv.Itoa(int(r.room_id))
			r.lobby.broadcastMessage(lobbyMessage{Sender: "room-" + room_id_string, Kind: "room-joined-player", Data: client_id_string})
			return
		}
		join_as_player = false
	}
	if c.lobby_room != nil {
		if c.lobby_room.room_id == r.room_id {
			c.send_message <- lobbyMessage{Sender: "server", Kind: "room-join-failed", Content: "Already in room"}
			return
		}
		c.lobby_room.removeClient(c, true)
	}
	if join_as_player && !r.spaceForPlayer() {
		join_as_player = false
	}
	if !join_as_player && !r.spaceForViewer() {
		c.send_message <- lobbyMessage{Sender: "server", Kind: "room-join-failed", Content: "No space in room"}
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
	client_id_string := strconv.Itoa(int(c.client_id))
	room_id_string := strconv.Itoa(int(r.room_id))
	message_kind := "room-joined-player"
	if !join_as_player {
		message_kind = "room-joined-viewer"
		if !r.gameNil() {
			r.game.GetBase().Viewers[c.client_id] = game.CreateViewer(c.client_id, c.nickname)
		}
	}
	r.lobby.broadcastMessage(lobbyMessage{Sender: "room-" + room_id_string, Kind: message_kind, Data: client_id_string})
}

func (r *LobbyRoom) spaceForPlayer() bool {
	return r.game_settings != nil && int(r.game_settings.MaxPlayers) > len(r.players)
}

func (r *LobbyRoom) spaceForViewer() bool {
	return r.game_settings != nil && int(r.game_settings.MaxViewers) > len(r.viewers)
}

func (r *LobbyRoom) removeClient(c *Client, client_leaves bool) {
	if r.host != nil && r.host.client_id == c.client_id && !r.gameStarted() {
		// TODO: make someone else the room host if game not started
		r.lobby.removeRoom(r)
		return
	}
	if r.gameStarted() {
		r.playerDisconnected(c)
	}
	if !r.gameStarted() || client_leaves {
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
	client_id_string := strconv.Itoa(int(c.client_id))
	room_id_string := strconv.Itoa(int(r.room_id))
	r.lobby.broadcastMessage(lobbyMessage{Sender: "room-" + room_id_string, Kind: "room-left", Data: client_id_string})
}

/** This needs to be the way we disconnect players from a game */
func (r *LobbyRoom) playerDisconnected(c *Client) {
	if game.Game_PlayerDisconnected(r.game, c.client_id) {
		delete_room := time.NewTimer(45 * time.Second)
		r.delete_timer = delete_room
		go func() {
			<-delete_room.C
			r.lobby.removeRoom(r)
		}()
	}
	client_id_string := strconv.Itoa(int(c.client_id))
	room_id_string := strconv.Itoa(int(r.room_id))
	r.broadcastMessage(lobbyMessage{Sender: "room-" + room_id_string, Kind: "game-player-disconnected", Data: client_id_string})
}

func (r *LobbyRoom) kickClient(c *Client) {
	if r.host.client_id == c.client_id {
		return
	}
	if c.lobby_room != nil && c.lobby_room.room_id == r.room_id {
		c.lobby_room = nil
	}
	delete(r.players, c.client_id)
	delete(r.viewers, c.client_id)
	client_id_string := strconv.Itoa(int(c.client_id))
	room_id_string := strconv.Itoa(int(r.room_id))
	r.lobby.broadcastMessage(lobbyMessage{Sender: "room-" + room_id_string, Kind: "room-kicked", Data: client_id_string})
}

func (r *LobbyRoom) promotePlayer(c *Client) {
	if r.host.client_id == c.client_id {
		return
	}
	if c.lobby_room == nil || c.lobby_room.room_id != r.room_id {
		return
	}
	r.host = c
	client_id_string := strconv.Itoa(int(c.client_id))
	room_id_string := strconv.Itoa(int(r.room_id))
	r.lobby.broadcastMessage(lobbyMessage{Sender: "room-" + room_id_string, Kind: "room-promoted", Data: client_id_string})
}

func (r *LobbyRoom) setViewer(c *Client) {
	if r.host.client_id == c.client_id || r.players[c.client_id] == nil {
		return
	}
	delete(r.players, c.client_id)
	r.viewers[c.client_id] = c
	client_id_string := strconv.Itoa(int(c.client_id))
	room_id_string := strconv.Itoa(int(r.room_id))
	r.lobby.broadcastMessage(lobbyMessage{Sender: "room-" + room_id_string, Kind: "room-viewer-set", Data: client_id_string})
}

func (r *LobbyRoom) setPlayer(c *Client) {
	if r.host.client_id == c.client_id || r.viewers[c.client_id] == nil {
		return
	}
	delete(r.viewers, c.client_id)
	r.players[c.client_id] = c
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
	r.game_settings = s
	room_id_string := strconv.Itoa(int(r.room_id))
	r.lobby.broadcastMessage(lobbyMessage{
		Sender: "room-" + room_id_string,
		Kind:   "room-settings-updated",
		Data:   string(settings_stringified),
	})
}

func (r *LobbyRoom) launchGame(game_id uint64) (game.Game, error) {
	launchable, launchable_error := r.launchable()
	if !launchable {
		message := fmt.Sprintf("Cannot launch game of id %d: %s", game_id, launchable_error)
		fmt.Fprintln(os.Stderr, message)
		return nil, errors.New(message)
	}
	base_game := game.CreateBaseGame(game_id, r.game_settings.GameType, r.game_settings.GameSpecificSettings)
	for _, player := range r.players {
		game.CreatePlayer(player.client_id, player.nickname, base_game)
	}
	for _, viewer := range r.viewers {
		base_game.Viewers[viewer.client_id] = game.CreateViewer(viewer.client_id, viewer.nickname)
	}
	var err error = nil
	var new_game game.Game = nil
	switch r.game_settings.GameType {
	case 1:
		new_game, err = fiddlesticks.CreateGame(base_game, r.PlayerAction)
	case 2:
		new_game, err = euchre.CreateGame(base_game)
	case 3:
		new_game, err = risq.CreateGame(base_game)
	case 4:
		new_game, err = test_game.CreateGame(base_game)
	default:
		err = fmt.Errorf(fmt.Sprintf("GameType not recognized: %d", r.game_settings.GameType))
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return nil, err
	}
	r.game = new_game
	room_id_string := strconv.Itoa(int(r.room_id))
	game_id_string := strconv.Itoa(int(game_id))
	r.lobby.broadcastMessage(lobbyMessage{Sender: "room-" + room_id_string, Kind: "room-launched", Data: game_id_string})
	return r.game, nil
}

func (r *LobbyRoom) GetGame() game.Game {
	return r.game
}

func (r *LobbyRoom) launchable() (bool, string) {
	if !r.valid() {
		return false, "Room invalid"
	}
	if r.game_settings == nil {
		return false, "No game settings"
	}
	settings_launchable, settings_launchable_error := r.game_settings.Launchable()
	if !settings_launchable {
		return false, settings_launchable_error
	}
	if !r.gameNil() {
		return false, "Game already launched"
	}
	return true, ""
}

func (r *LobbyRoom) valid() bool {
	if r.room_id < 1 {
		return false
	}
	if !r.gameNil() && !r.game.Valid() {
		return false
	}
	if r.gameNil() && (r.host == nil || !r.host.validDebug(true)) {
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

func (r *LobbyRoom) ToFrontend() gin.H {
	room := gin.H{
		"room_id":          strconv.Itoa(int(r.room_id)),
		"room_name":        r.room_name,
		"room_description": r.room_description,
		"game_settings":    r.game_settings.ToFrontend(),
	}
	if r.host != nil {
		room["host"] = r.host.ToFrontend()
	}
	players := []gin.H{}
	for _, player := range r.players {
		if player != nil {
			players = append(players, player.ToFrontend())
		}
	}
	room["players"] = players
	viewers := []gin.H{}
	for _, viewer := range r.viewers {
		if viewer != nil {
			viewers = append(viewers, viewer.ToFrontend())
		}
	}
	room["viewers"] = viewers
	if !r.gameNil() {
		room["game_id"] = strconv.Itoa(int(game.Game_GetId(r.game)))
	}
	return room
}
