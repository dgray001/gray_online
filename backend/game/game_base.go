package game

import (
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
)

type UpdateMessage struct {
	Id      int
	Kind    string
	Content gin.H
}

func (u *UpdateMessage) toFrontend() gin.H {
	return gin.H{
		"update_id": u.Id,
		"kind":      u.Kind,
		"content":   u.Content,
	}
}

type GameBase struct {
	Game_id   uint64
	game_type uint8
	// here the map keys are the lobby client ids
	Players              map[uint64]*Player
	Viewers              map[uint64]*Viewer
	game_started         bool
	game_ended           bool
	player_updates       []*PlayerAction
	ViewerUpdates        chan *UpdateMessage
	viewer_update_list   []*UpdateMessage
	GameSpecificSettings map[string]interface{}
	GameEndedChannel     chan string
}

func CreateBaseGame(game_id uint64, game_type uint8, game_specific_settings map[string]interface{}) *GameBase {
	return &GameBase{
		Game_id:              game_id,
		game_type:            game_type,
		Players:              make(map[uint64]*Player),
		Viewers:              make(map[uint64]*Viewer),
		game_started:         false,
		game_ended:           false,
		player_updates:       make([]*PlayerAction, 0),
		ViewerUpdates:        make(chan *UpdateMessage),
		viewer_update_list:   make([]*UpdateMessage, 0),
		GameSpecificSettings: game_specific_settings,
		GameEndedChannel:     make(chan string),
	}
}

// Returns whether this connection should trigger the game to start
func (g *GameBase) PlayerConnected(client_id uint64) bool {
	player := g.Players[client_id]
	if player == nil || player.connected {
		return false
	}
	player.connected = true
	if g.game_started {
		return false
	}
	for _, player := range g.Players {
		if !player.connected {
			return false
		}
	}
	return true
}

// Returns whether this disconnect should trigger the game to end
func (g *GameBase) PlayerDisconnected(client_id uint64) bool {
	player := g.Players[client_id]
	if player == nil || !player.connected {
		return false
	}
	player.connected = false
	for _, player := range g.Players {
		if player.connected {
			return false
		}
	}
	return true
}

func (g *GameBase) GameStarted() bool {
	return g.game_started
}

func (g *GameBase) StartGame() {
	if g.game_started {
		fmt.Fprintln(os.Stderr, "Game already started")
	}
	g.game_started = true
}

func (g *GameBase) AddAction(action PlayerAction) PlayerAction {
	action.action_id = len(g.player_updates) + 1 // start at 1
	g.player_updates = append(g.player_updates, &action)
	return action
}

func (g *GameBase) AddViewerUpdate(update *UpdateMessage) {
	if !g.game_started {
		fmt.Fprintln(os.Stderr, "Can't add update to game that isn't started")
		return
	}
	if g.game_ended {
		fmt.Fprintln(os.Stderr, "Can't add update to game that is ended")
		return
	}
	update.Id = len(g.viewer_update_list) + 1 // start at 1
	g.viewer_update_list = append(g.viewer_update_list, update)
	g.ViewerUpdates <- update
	for _, viewer := range g.Viewers {
		if viewer == nil || !viewer.connected {
			continue
		}
		viewer.Updates <- update
	}
}

func (g *GameBase) ResendPlayerUpdate(client_id uint64, update_id int) {
	player := g.Players[client_id]
	if player == nil || update_id < 1 || update_id > len(player.update_list) {
		return
	}
	player.Updates <- player.update_list[update_id-1]
}

func (g *GameBase) ResendViewerUpdate(client_id uint64, update_id int) {
	viewer := g.Viewers[client_id]
	if viewer == nil || update_id < 1 || update_id > len(g.viewer_update_list) {
		return
	}
	viewer.Updates <- g.viewer_update_list[update_id-1]
}

func (g *GameBase) ResendLastUpdate(client_id uint64) {
	player := g.Players[client_id]
	if player == nil || len(player.update_list) < 1 {
		viewer := g.Viewers[client_id]
		if viewer != nil && len(g.viewer_update_list) > 0 {
			viewer.Updates <- g.viewer_update_list[len(g.viewer_update_list)-1]
		}
		return
	}
	player.Updates <- player.update_list[len(player.update_list)-1]
}

func (g *GameBase) GameEnded() bool {
	return g.game_ended
}

func (g *GameBase) EndGame(message string) {
	if g.game_ended {
		fmt.Fprintln(os.Stderr, "Game already ended")
	}
	g.game_ended = true
	g.GameEndedChannel <- message
}

func (g *GameBase) ToFrontend(client_id uint64, is_viewer bool) gin.H {
	game_base := gin.H{
		"game_id":      g.Game_id,
		"game_type":    g.game_type,
		"game_started": g.game_started,
		"game_ended":   g.game_ended,
	}
	players := []gin.H{}
	for _, player := range g.Players {
		if player != nil {
			players = append(players, player.ToFrontend(is_viewer || client_id == player.client_id))
		}
	}
	game_base["players"] = players
	viewers := []gin.H{}
	for _, viewer := range g.Viewers {
		if viewer != nil {
			viewers = append(viewers, viewer.ToFrontend())
		}
	}
	game_base["viewers"] = viewers
	if is_viewer {
		player_actions := []gin.H{}
		for _, player_action := range g.player_updates {
			if player_action != nil {
				player_actions = append(player_actions, player_action.toFrontend())
			}
		}
		game_base["player_actions"] = player_actions
		viewer_updates := []gin.H{}
		for _, viewer_update := range g.viewer_update_list {
			if viewer_update != nil {
				viewer_updates = append(viewer_updates, viewer_update.toFrontend())
			}
		}
		game_base["viewer_updates"] = viewer_updates
	}
	return game_base
}
