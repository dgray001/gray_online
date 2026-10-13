package game

import (
	"fmt"
	"os"
	"sync"

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
	game_type GameType
	// here the map keys are the lobby client ids
	Players map[uint64]*Player
	Viewers map[uint64]*Viewer
	// the game base should maintain a separate ai id counter
	next_ai_id            uint32
	AiPlayers             map[uint32]*Player
	game_started          bool
	game_ended            bool
	player_updates        []*PlayerAction
	ViewerUpdates         chan *UpdateMessage
	next_viewer_update_id int
	viewer_update_list    []*UpdateMessage
	viewer_update_list_mu sync.RWMutex
	GameSpecificSettings  map[string]interface{}
	GameEndedChannel      chan string
	// set by the owning room at launch; routes a state read onto its actor goroutine
	RequestToFrontend func(client_id uint64, is_viewer bool) (gin.H, error)
	// both set by the owning room: run a task on its actor goroutine, and read game state there without a round trip
	RunInRoom    func(task func())
	CurrentState func(client_id uint64, is_viewer bool) gin.H
}

func CreateBaseGame(game_id uint64, game_type GameType, game_specific_settings map[string]interface{}) *GameBase {
	return &GameBase{
		Game_id:               game_id,
		game_type:             game_type,
		Players:               make(map[uint64]*Player),
		Viewers:               make(map[uint64]*Viewer),
		next_ai_id:            1,
		AiPlayers:             make(map[uint32]*Player),
		game_started:          false,
		game_ended:            false,
		player_updates:        make([]*PlayerAction, 0),
		ViewerUpdates:         make(chan *UpdateMessage, 12),
		next_viewer_update_id: 1,
		viewer_update_list:    make([]*UpdateMessage, 0),
		GameSpecificSettings:  game_specific_settings,
		GameEndedChannel:      make(chan string, 4),
	}
}

func (g *GameBase) PersistantHistory() bool {
	switch g.game_type {
	case GameType_RISQ:
		return false
	default:
		return true
	}
}

func (g *GameBase) NextAiId() uint32 {
	next_ai_id := g.next_ai_id
	g.next_ai_id++
	return next_ai_id
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
		return
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
	g.viewer_update_list_mu.Lock()
	if !g.PersistantHistory() {
		g.viewer_update_list = make([]*UpdateMessage, 0)
	}
	// copy so this Id assignment can't race other recipients' AddUpdate/AddViewerUpdate calls on the shared update
	own_update := *update
	own_update.Id = g.next_viewer_update_id
	g.next_viewer_update_id++
	g.viewer_update_list = append(g.viewer_update_list, &own_update)
	g.viewer_update_list_mu.Unlock()
	select {
	case g.ViewerUpdates <- &own_update:
	default:
		fmt.Fprintln(os.Stderr, "Dropped viewer update - ViewerUpdates buffer full:", own_update.Kind)
	}
	for _, viewer := range g.Viewers {
		if viewer == nil || !viewer.connected {
			continue
		}
		select {
		case viewer.Updates <- &own_update:
		default:
			fmt.Fprintln(os.Stderr, "Dropped update to viewer", viewer.client_id, "- update buffer full:", own_update.Kind)
		}
	}
}

func (g *GameBase) HasViewers() bool {
	return len(g.Viewers) > 0
}

func (g *GameBase) DropViewerUpdateHistory() {
	g.viewer_update_list_mu.Lock()
	g.viewer_update_list = make([]*UpdateMessage, 0)
	g.viewer_update_list_mu.Unlock()
}

func (g *GameBase) ResendPlayerUpdate(client_id uint64, update_id int) {
	if !g.PersistantHistory() {
		g.ResendLastUpdate(client_id)
		return
	}
	player := g.Players[client_id]
	if player == nil {
		return
	}
	player.update_list_mu.RLock()
	if update_id < 1 || update_id > len(player.update_list) {
		player.update_list_mu.RUnlock()
		return
	}
	update := player.update_list[update_id-1]
	player.update_list_mu.RUnlock()
	select {
	case player.Updates <- update:
	default:
		fmt.Fprintln(os.Stderr, "Dropped resend update to player", player.Player_id, "- update buffer full")
	}
}

func (g *GameBase) ResendViewerUpdate(client_id uint64, update_id int) {
	if !g.PersistantHistory() {
		g.ResendLastUpdate(client_id)
		return
	}
	viewer := g.Viewers[client_id]
	if viewer == nil {
		return
	}
	g.viewer_update_list_mu.RLock()
	if update_id < 1 || update_id > len(g.viewer_update_list) {
		g.viewer_update_list_mu.RUnlock()
		return
	}
	update := g.viewer_update_list[update_id-1]
	g.viewer_update_list_mu.RUnlock()
	select {
	case viewer.Updates <- update:
	default:
		fmt.Fprintln(os.Stderr, "Dropped resend update to viewer", viewer.client_id, "- update buffer full")
	}
}

func (g *GameBase) ResendLastUpdate(client_id uint64) {
	if player := g.Players[client_id]; player != nil && g.regenerateThenReplay(player, client_id) {
		return
	}
	g.replayLastUpdate(client_id)
}

// The newest update id sent to this client and whether its stored snapshot is stale; ok is false for unknown clients
func (g *GameBase) LatestUpdate(client_id uint64) (id int, stale bool, ok bool) {
	if player := g.Players[client_id]; player != nil {
		id, stale = player.latestUpdate()
		return id, stale, true
	}
	if g.Viewers[client_id] != nil {
		g.viewer_update_list_mu.RLock()
		defer g.viewer_update_list_mu.RUnlock()
		return g.next_viewer_update_id - 1, false, true
	}
	return 0, false, false
}

// Replays a stale snapshot only after regenerating it on the room goroutine; false means nothing was scheduled
func (g *GameBase) regenerateThenReplay(player *Player, client_id uint64) bool {
	if g.RunInRoom == nil || g.CurrentState == nil || !player.snapshotIsStale() {
		return false
	}
	g.RunInRoom(func() {
		if state := g.CurrentState(client_id, false); state != nil {
			player.refreshStaleSnapshot(state)
		}
		g.replayLastUpdate(client_id)
	})
	return true
}

func (g *GameBase) replayLastUpdate(client_id uint64) {
	fmt.Println("Resending last update for client", client_id)
	if player := g.Players[client_id]; player != nil {
		player.update_list_mu.RLock()
		if len(player.update_list) > 0 {
			update := player.update_list[len(player.update_list)-1]
			player.update_list_mu.RUnlock()
			select {
			case player.Updates <- update:
			default:
				fmt.Fprintln(os.Stderr, "Dropped resend update to player", player.Player_id, "- update buffer full")
			}
			return
		}
		player.update_list_mu.RUnlock()
	}
	if viewer := g.Viewers[client_id]; viewer != nil {
		g.viewer_update_list_mu.RLock()
		if len(g.viewer_update_list) > 0 {
			update := g.viewer_update_list[len(g.viewer_update_list)-1]
			g.viewer_update_list_mu.RUnlock()
			select {
			case viewer.Updates <- update:
			default:
				fmt.Fprintln(os.Stderr, "Dropped resend update to viewer", viewer.client_id, "- update buffer full")
			}
		} else {
			g.viewer_update_list_mu.RUnlock()
		}
	}
}

func (g *GameBase) GameEnded() bool {
	return g.game_ended
}

func (g *GameBase) EndGame(message string) {
	if g.game_ended {
		fmt.Fprintln(os.Stderr, "Game already ended")
		return
	}
	g.game_ended = true
	g.GameEndedChannel <- message
}

func (g *GameBase) ToFrontend(client_id uint64, is_viewer bool) gin.H {
	latest_update_id := 0
	if is_viewer {
		latest_update_id = g.next_viewer_update_id - 1
	} else if player := g.Players[client_id]; player != nil {
		latest_update_id = player.next_update_id - 1
	}
	if latest_update_id < 0 {
		latest_update_id = 0
	}
	game_base := gin.H{
		"game_id":            g.Game_id,
		"game_type":          g.game_type,
		"game_started":       g.game_started,
		"game_ended":         g.game_ended,
		"persistant_history": g.PersistantHistory(),
		"latest_update_id":   latest_update_id,
	}
	players := []gin.H{}
	for _, player := range g.Players {
		if player != nil {
			players = append(players, player.ToFrontend(g.PersistantHistory() && (is_viewer || client_id == player.client_id)))
		}
	}
	for _, player := range g.AiPlayers {
		if player != nil {
			players = append(players, player.ToFrontend(g.PersistantHistory() && is_viewer))
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
		g.viewer_update_list_mu.RLock()
		for _, viewer_update := range g.viewer_update_list {
			if viewer_update != nil {
				viewer_updates = append(viewer_updates, viewer_update.toFrontend())
			}
		}
		g.viewer_update_list_mu.RUnlock()
		game_base["viewer_updates"] = viewer_updates
	}
	return game_base
}
