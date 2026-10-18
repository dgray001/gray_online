package game

import (
	"maps"
	"sync"

	"github.com/dgray001/gray_online/util"
	"github.com/gin-gonic/gin"
)

// Content key holding a full game-state snapshot in updates of games with non-persistent history
const snapshotContentKey = "game"

// Content key holding the recipient's setting revision in each setting update
const settingsRevContentKey = "rev"

type PlayerAction struct {
	action_id int
	Client_id int
	Kind      string
	Action    gin.H
	Ai_id     int
}

func (a *PlayerAction) toFrontend() gin.H {
	return gin.H{
		"action_id": a.action_id,
		"client_id": a.Client_id,
		"kind":      a.Kind,
		"action":    a.Action,
		"ai_id":     a.Ai_id,
	}
}

type Player struct {
	client_id       uint64
	ai_player_id    uint32 // specific to the game
	ai_running      bool   // can be true for human players
	Player_id       int
	nickname        string
	connected       bool
	Updates         chan *UpdateMessage
	FailedUpdates   chan *UpdateMessage
	SettingUpdates  chan *UpdateMessage
	AiUpdates       chan *UpdateMessage
	AiFailedUpdates chan *UpdateMessage
	next_update_id  int
	forwarded_id    int
	update_list     []*UpdateMessage
	update_list_mu  sync.RWMutex
	snapshot_stale  bool
	settings_rev    int
	base_game       *GameBase
}

func (p *Player) errLogger() *util.Logger {
	if p.base_game != nil && p.base_game.Err != nil {
		return p.base_game.Err
	}
	return util.DefaultErr
}

func (p *Player) stdLogger() *util.Logger {
	if p.base_game != nil && p.base_game.Std != nil {
		return p.base_game.Std
	}
	return util.DefaultStd
}

const UPDATE_CHANNEL_SIZE = 8

func CreatePlayer(client_id uint64, nickname string, base_game *GameBase) {
	player := &Player{
		client_id:       client_id,
		ai_player_id:    0,
		ai_running:      false,
		Player_id:       -1,
		nickname:        nickname,
		connected:       false,
		Updates:         make(chan *UpdateMessage, UPDATE_CHANNEL_SIZE),
		AiUpdates:       make(chan *UpdateMessage, UPDATE_CHANNEL_SIZE),
		FailedUpdates:   make(chan *UpdateMessage, UPDATE_CHANNEL_SIZE),
		SettingUpdates:  make(chan *UpdateMessage, UPDATE_CHANNEL_SIZE),
		AiFailedUpdates: make(chan *UpdateMessage, UPDATE_CHANNEL_SIZE),
		next_update_id:  1,
		update_list:     []*UpdateMessage{},
		base_game:       base_game,
	}
	base_game.Players[client_id] = player
}

func CreateAiPlayer(nickname string, base_game *GameBase) *Player {
	ai_id := base_game.NextAiId()
	player := &Player{
		client_id:       0,
		ai_player_id:    ai_id,
		ai_running:      true,
		Player_id:       -1,
		nickname:        nickname,
		connected:       false,
		Updates:         make(chan *UpdateMessage, UPDATE_CHANNEL_SIZE),
		AiUpdates:       make(chan *UpdateMessage, UPDATE_CHANNEL_SIZE),
		FailedUpdates:   make(chan *UpdateMessage, UPDATE_CHANNEL_SIZE),
		SettingUpdates:  make(chan *UpdateMessage, UPDATE_CHANNEL_SIZE),
		AiFailedUpdates: make(chan *UpdateMessage, UPDATE_CHANNEL_SIZE),
		next_update_id:  1,
		update_list:     []*UpdateMessage{},
		base_game:       base_game,
	}
	base_game.AiPlayers[ai_id] = player
	return player
}

// Whether an update sent now could be read: AIs always read, humans only while connected
func (p *Player) HasRecipient() bool {
	return p.ai_running || p.connected
}

// Forgets the stored history so a resend can't replay a snapshot older than the current state
func (p *Player) DropUpdateHistory() {
	p.update_list_mu.Lock()
	p.update_list = make([]*UpdateMessage, 0)
	p.update_list_mu.Unlock()
}

// Marks the newest stored snapshot as outdated, to be regenerated lazily on its next replay
func (p *Player) MarkSnapshotStale() {
	p.update_list_mu.Lock()
	p.snapshot_stale = true
	p.update_list_mu.Unlock()
}

func (p *Player) snapshotIsStale() bool {
	p.update_list_mu.RLock()
	defer p.update_list_mu.RUnlock()
	return p.snapshot_stale && len(p.update_list) > 0
}

// The id of the newest update queued on this player's socket (or dropped) and the revision of their newest setting update
func (p *Player) latestUpdate() (int, int) {
	p.update_list_mu.RLock()
	defer p.update_list_mu.RUnlock()
	return p.forwarded_id, p.settings_rev
}

// Must be called only after the update is queued on the socket, so a lookup reply can never overtake it
func (p *Player) MarkForwarded(update_id int) {
	p.update_list_mu.Lock()
	defer p.update_list_mu.Unlock()
	p.forwarded_id = max(p.forwarded_id, update_id)
}

// Counts every setting update made for this player, delivered or not, so a client can detect a missed one
func (p *Player) SettingsRev() int {
	p.update_list_mu.RLock()
	defer p.update_list_mu.RUnlock()
	return p.settings_rev
}

// Swaps in current state for a stale newest snapshot; a newer update that already replaced it needs nothing
func (p *Player) refreshStaleSnapshot(state gin.H) {
	p.update_list_mu.Lock()
	defer p.update_list_mu.Unlock()
	if !p.snapshot_stale || len(p.update_list) == 0 {
		return
	}
	last := *p.update_list[len(p.update_list)-1]
	last.Content = maps.Clone(last.Content)
	last.Content[snapshotContentKey] = state
	p.update_list[len(p.update_list)-1] = &last
	p.snapshot_stale = false
}

func (p *Player) IsHumanPlayer() bool {
	return p.client_id > 0
}

func (p *Player) RunningAi() {
	p.ai_running = true
}

func (p *Player) AddUpdate(update *UpdateMessage) {
	if !p.base_game.game_started {
		util.Error(p.errLogger(), "Can't add update to game that isn't started")
		return
	}
	if p.base_game.game_ended {
		util.Error(p.errLogger(), "Can't add update to game that is ended")
		return
	}
	p.update_list_mu.Lock()
	if !p.base_game.PersistantHistory() {
		p.update_list = make([]*UpdateMessage, 0)
	}
	// copy so this recipient's Id assignment can't race other recipients' AddUpdate/AddViewerUpdate calls on the shared update
	own_update := *update
	own_update.Id = p.next_update_id
	p.next_update_id++
	p.update_list = append(p.update_list, &own_update)
	p.snapshot_stale = false
	p.update_list_mu.Unlock()
	if p.IsHumanPlayer() {
		select {
		case p.Updates <- &own_update:
		default:
			util.Errorf(p.errLogger(), "Dropped update to player %d - update buffer full: %s", p.Player_id, own_update.Kind)
			p.MarkForwarded(own_update.Id)
		}
	}
	if p.ai_running {
		select {
		case p.AiUpdates <- &own_update:
		default:
			// evict the oldest wakeup so the newest (e.g. a new turn) is never the one lost
			select {
			case evicted := <-p.AiUpdates:
				util.Errorf(p.errLogger(), "Evicted update for AI player %d - buffer full: %s %d for %s", p.Player_id, evicted.Kind, evicted.Id, own_update.Kind)
			default:
			}
			select {
			case p.AiUpdates <- &own_update:
			default:
				util.Errorf(p.errLogger(), "Dropped update to AI player %d - update buffer full: %s", p.Player_id, own_update.Kind)
			}
		}
	}
}

// ReplayUpdates calls apply, in order, for every update after last_applied_id in this player's
// permanent update history, returning the new last_applied_id. This is for games whose AI
// decision loop wakes up on an AiUpdates channel receive but shouldn't trust that channel to
// deliver every update exactly once (AddUpdate drops updates non-blocking when that channel's
// buffer is full) — walking the permanent history instead makes a dropped wakeup harmless, the
// same way the frontend replays from its own update history by id. Only meaningful for games with
// GameBase.PersistantHistory() true; a non-persistent game's update_list is reset on every update.
func (p *Player) ReplayUpdates(last_applied_id int, apply func(*UpdateMessage)) int {
	p.update_list_mu.RLock()
	defer p.update_list_mu.RUnlock()
	for last_applied_id < len(p.update_list) {
		apply(p.update_list[last_applied_id])
		last_applied_id++
	}
	return last_applied_id
}

// Delivers a small per-player settings acknowledgement outside the sequenced, replayable update history
func (p *Player) AddSettingUpdate(update *UpdateMessage) {
	if !p.IsHumanPlayer() {
		return
	}
	p.update_list_mu.Lock()
	p.settings_rev++
	rev := p.settings_rev
	p.update_list_mu.Unlock()
	if !p.connected {
		return
	}
	own_update := *update
	own_update.Content = make(gin.H, len(update.Content)+1)
	maps.Copy(own_update.Content, update.Content)
	own_update.Content[settingsRevContentKey] = rev
	select {
	case p.SettingUpdates <- &own_update:
	default:
		util.Errorf(p.errLogger(), "Dropped setting update to player %d - buffer full: %s", p.Player_id, update.Kind)
	}
}

func (p *Player) AddFailedUpdate(update *UpdateMessage) {
	if p.IsHumanPlayer() {
		select {
		case p.FailedUpdates <- update:
		default:
			util.Errorf(p.errLogger(), "Dropped failed update to player %d - buffer full: %s", p.Player_id, update.Kind)
		}
	}
	if p.ai_running {
		select {
		case p.AiFailedUpdates <- update:
		default:
			util.Errorf(p.errLogger(), "Dropped failed update to AI player %d - buffer full: %s", p.Player_id, update.Kind)
		}
	}
}

func (p *Player) AddFailedUpdateShorthand(kind string, message string) {
	util.Info(p.stdLogger(), message)
	p.AddFailedUpdate(&UpdateMessage{Kind: kind, Content: gin.H{"message": message, "player_id": p.Player_id}})
}

func (p *Player) GetNickname() string {
	return p.nickname
}

func (p *Player) GetConnected() bool {
	return p.connected
}

func (p *Player) GetClientId() uint64 {
	return p.client_id
}

func (p *Player) GetBase() *GameBase {
	return p.base_game
}

func (p *Player) GetAiId() uint32 {
	return p.ai_player_id
}

func (p *Player) ToFrontend(show_updates bool) gin.H {
	player := gin.H{
		"client_id":    p.client_id,
		"ai_player_id": p.ai_player_id,
		"player_id":    p.Player_id,
		"nickname":     p.nickname,
		"connected":    p.connected,
	}
	if show_updates {
		updates := []gin.H{}
		p.update_list_mu.RLock()
		for _, update := range p.update_list {
			if update != nil {
				updates = append(updates, update.toFrontend())
			}
		}
		p.update_list_mu.RUnlock()
		player["updates"] = updates
	}
	return player
}
