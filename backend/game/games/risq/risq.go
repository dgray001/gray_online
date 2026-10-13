package risq

import (
	"encoding/json"
	"fmt"
	"maps"
	"math/rand"
	"sync"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/util"
	"github.com/gin-gonic/gin"
)

/*
   ================
   >>>>> RISQ <<<<<
   ================

   Objective: Build your empire and conquer the world!
   Description: Strategy board game with simultaneous turn resolution, hexagonal
     map, complex deterministic mechanics (no randomness after map generation),
     resource gathering, empire-building, complex combat, and medieval themes.
*/

type GameRisq struct {
	tick_history *turnJournal

	metrics                   gameMetrics
	game                      *game.GameBase
	outcome                   *risqOutcome
	players                   []*RisqPlayer
	board_size                uint16
	visibility_mode           defs.VisibilityMode
	population_limit          uint16
	background_image          string
	background_top_left       game_utils.Coordinate2D
	background_top_right      game_utils.Coordinate2D
	spaces                    [][]*RisqSpace
	units                     map[uint64]*RisqUnit
	buildings                 map[uint64]*RisqBuilding
	next_resource_internal_id uint64
	next_building_internal_id uint64
	next_unit_internal_id     uint64
	next_order_internal_id    uint64
	turn_number               uint16
	current_tick              uint16
	// True if waiting for players to give orders and false if resolving active orders
	giving_orders bool
	// Recomputed each tick: contested resources are water-filled instead of first-come-first-served
	gather_allotments map[*RisqUnit]float64
	// Recomputed each tick: settles which simultaneous garrison attempts get a building's remaining slots
	garrison_allotments  map[*RisqUnit]bool
	production_garrisons map[*RisqBuilding]bool
	// Recomputed each tick: settles which unit founds a new building when several race for the same empty zone
	construction_winners map[*RisqZone]uint64
	// Recomputed each tick: building id each winning zone's new foundation takes
	foundation_ids map[*RisqZone]uint64
	// Recomputed each tick: settles which building's unit production completes when several race for the last population slots
	population_slot_winners map[*RisqBuilding]bool
	// Recomputed each tick: a player's simultaneous repairs are water-filled per resource category
	repair_allotments map[*RisqUnit]float64
	// Recomputed each tick: assigns new unit internal_ids by producing building's own internal_id
	unit_creation_ids map[*RisqBuilding]uint64
	// Recomputed each tick: snapshots population-capped status before any of this tick's completions
	population_capped map[int]bool
	// Zones whose cosmetic terrain_override should clear at the start of cleanupDeleted's NEXT call
	pending_terrain_clears []*RisqZone
	// Techs finishing production this tick; applied after this tick's health deltas
	pending_tech_completions []techCompletion
	completed_gatherables    []*RisqBuilding
	regions                  []*RisqRegion
	space_distances          *spaceDistances
	space_links              []gin.H
	mercenaries_need_region  bool
	rng                      *rand.Rand
	ai_goroutines            sync.WaitGroup
}

func (r *GameRisq) nextResourceInternalId() uint64 {
	r.next_resource_internal_id++
	return r.next_resource_internal_id
}

func (r *GameRisq) nextBuildingInternalId() uint64 {
	r.next_building_internal_id++
	return r.next_building_internal_id
}

func (r *GameRisq) nextUnitInternalId() uint64 {
	r.next_unit_internal_id++
	return r.next_unit_internal_id
}

func (r *GameRisq) nextOrderInternalId() uint64 {
	r.next_order_internal_id++
	return r.next_order_internal_id
}

func (r *GameRisq) GetBase() *game.GameBase {
	return r.game
}

func (r *GameRisq) StartGame() {
	r.recalculateOwnership()
	r.recalculateVision()
	r.refreshScores()
	r.updateEliminated()
	r.checkWinCondition()
	r.metrics.recordTurn(r)
	if !r.game.GameEnded() {
		r.startNextTurn()
	}
}

func (r *GameRisq) endTurn() {
	defer util.PerfTimer("end_turn", map[string]any{"turn": r.turn_number})()
	r.recalculateOwnership()
	r.payGoldIncome()
	r.resolvePendingMercenaries()
	r.recalculateVision()
	r.refreshScores()
	r.updateEliminated()
	r.finalizeTurnReports()
	r.metrics.recordTurn(r)
	r.logStateHash("end")
}

func (r *GameRisq) startNextTurn() {
	defer util.PerfTimer("start_next_turn", map[string]any{"turn": r.turn_number + 1})()
	r.metrics.recordRefresh(r)
	r.recordEconomyStamina()
	r.turn_number++
	for _, player := range r.players {
		player.orders_submitted = false
	}
	for o := range r.allOrderables() {
		var base *orderableBase
		switch actor := o.(type) {
		case *RisqUnit:
			base = &actor.orderableBase
		case *RisqBuilding:
			base = &actor.orderableBase
		}
		before := base.current_stamina
		o.refreshStamina()
		util.DebugLog.Printf("stamina turn=%d actor=%d:%d player=%d before=%d grant=%d after=%d",
			r.turn_number, o.OrderableType(), o.internalId(), base.player_id, before, base.turn_stamina, base.current_stamina)
	}
	r.beginTurnReports()
	r.giving_orders = true
	r.captureTickBoundary("start-turn")
	r.broadcastStartTurn()
}

func (r *GameRisq) broadcastStartTurn() {
	r.broadcastSnapshot("start-turn", gin.H{})
}

// Builds a player's full-state update only if someone can read it; otherwise drops the stale history
func (r *GameRisq) sendSnapshot(player *game.Player, kind string, content gin.H) {
	if !player.HasRecipient() {
		player.DropUpdateHistory()
		return
	}
	perf_fields := map[string]any{"turn": r.turn_number, "player": player.Player_id, "update_kind": kind}
	stop_perf := util.PerfTimer("snapshot_build", perf_fields)
	content["game"] = r.toFrontendFor(player.Player_id, player.GetClientId(), false)
	if util.PerfEnabled() {
		encoded, _ := json.Marshal(content)
		perf_fields["bytes"] = len(encoded)
	}
	stop_perf()
	player.AddUpdate(&game.UpdateMessage{Kind: kind, Content: content})
}

// Sends a small private settings acknowledgement; the stored snapshot is now outdated and regenerates on its next replay
func (r *GameRisq) sendSetting(player *game.Player, kind string, content gin.H) {
	player.MarkSnapshotStale()
	player.AddSettingUpdate(&game.UpdateMessage{Kind: kind, Content: content})
}

func (r *GameRisq) sendViewerSnapshot(kind string, content gin.H) {
	if !r.game.HasViewers() {
		r.game.DropViewerUpdateHistory()
		return
	}
	content["game"] = r.ToFrontend(0, true)
	r.game.AddViewerUpdate(&game.UpdateMessage{Kind: kind, Content: content})
}

func (r *GameRisq) broadcastSnapshot(kind string, extra gin.H) {
	for _, player := range r.players {
		r.sendSnapshot(player.player, kind, maps.Clone(extra))
	}
	r.sendViewerSnapshot(kind, maps.Clone(extra))
}

func (r *GameRisq) updateEliminated() {
	for _, player := range r.players {
		if player.eliminated {
			continue
		}
		if !player.canHaveUnits() {
			player.eliminated = true
			player.report.recordEliminated()
			player.stopAi()
		}
	}
}

// Ends every ai goroutine and waits for them, so none outlive the game; call whenever turns stop resolving
func (r *GameRisq) StopAi() {
	for _, player := range r.players {
		player.stopAi()
	}
	r.ai_goroutines.Wait()
}

func (r *GameRisq) checkWinCondition() {
	if len(r.players) < 2 {
		return
	}
	var remaining []*RisqPlayer
	for _, player := range r.players {
		if !player.eliminated {
			remaining = append(remaining, player)
		}
	}
	if len(remaining) > 1 {
		return
	}
	r.StopAi()
	r.clearAllOrders()
	r.outcome = &risqOutcome{winner_player_ids: make([]int, 0, len(remaining))}
	for _, player := range remaining {
		r.outcome.winner_player_ids = append(r.outcome.winner_player_ids, player.player.Player_id)
	}
	r.giving_orders = false
	r.broadcastStartTurn() // no further start-turn goes out, so clients need this final state
	if len(remaining) == 1 {
		r.game.EndGame(fmt.Sprintf("%s wins!", remaining[0].player.GetNickname()))
	} else {
		r.game.EndGame("No players remaining")
	}
}

func (r *GameRisq) Valid() bool {
	if r.game == nil {
		return false
	}
	for _, player := range r.players {
		if !player.valid() {
			return false
		}
	}
	return true
}

func (r *GameRisq) PlayerDisconnected(client_id uint64) {
}

func (r *GameRisq) PlayerReconnected(client_id uint64) {
}

func (r *GameRisq) ToFrontend(client_id uint64, is_viewer bool) gin.H {
	player_id := -1
	if !is_viewer {
		for id, player := range r.players {
			if player != nil && player.player.GetClientId() == client_id {
				player_id = id
				break
			}
		}
	}
	return r.toFrontendFor(player_id, client_id, is_viewer)
}

func (r *GameRisq) viewerPlayerId(client_id uint64) int {
	for id, player := range r.players {
		if player != nil && player.player.GetClientId() == client_id {
			return id
		}
	}
	return -1
}

// Just the players section of ToFrontend, for callers that skip building the board
func (r *GameRisq) PlayersToFrontend(client_id uint64) []gin.H {
	viewer := r.viewerPlayerId(client_id)
	players := []gin.H{}
	for _, player := range r.players {
		if player != nil {
			players = append(players, player.toFrontend(viewer))
		}
	}
	return players
}

// One player's entry of PlayersToFrontend as seen by client_id
func (r *GameRisq) PlayerToFrontend(client_id uint64, player_id int) gin.H {
	return r.players[player_id].toFrontend(r.viewerPlayerId(client_id))
}

// Keyed by player id since every AI player shares client id 0
func (r *GameRisq) toFrontendFor(player_id int, client_id uint64, is_viewer bool) gin.H {
	game := gin.H{
		"board_size":           r.board_size,
		"population_limit":     r.population_limit,
		"background_image":     r.background_image,
		"background_top_left":  []int{r.background_top_left.X, r.background_top_left.Y},
		"background_top_right": []int{r.background_top_right.X, r.background_top_right.Y},
		"turn_number":          r.turn_number,
		"giving_orders":        r.giving_orders,
	}
	if r.game != nil {
		game["game_base"] = r.game.ToFrontend(client_id, is_viewer)
	}
	if r.outcome != nil {
		game["outcome"] = r.outcome.toFrontend()
		game["game_base"].(gin.H)["game_ended"] = true
	}
	players := []gin.H{}
	for _, player := range r.players {
		if player != nil {
			players = append(players, player.toFrontend(player_id))
		}
	}
	game["players"] = players
	spaces := [][]gin.H{}
	for _, row := range r.spaces {
		spaces_row := []gin.H{}
		for _, space := range row {
			if space == nil {
				spaces_row = append(spaces_row, nil)
				continue
			}
			spaces_row = append(spaces_row, space.toFrontend(player_id, is_viewer))
		}
		spaces = append(spaces, spaces_row)
	}
	game["spaces"] = spaces
	game["space_links"] = r.space_links
	regions := []gin.H{}
	for _, region := range r.regions {
		if reg := region.toFrontend(r, player_id); reg != nil {
			regions = append(regions, reg)
		}
	}
	game["regions"] = regions
	game["last_turn_replay"] = nil
	if r.tick_history != nil && player_id >= 0 {
		game["last_turn_replay"] = r.tick_history.toFrontend(player_id)
	}
	return game
}
