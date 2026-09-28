package risq

import (
	"fmt"
	"math/rand"
	"sync"

	"github.com/dgray001/gray_online/game"
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
	game                      *game.GameBase
	players                   []*RisqPlayer
	board_size                uint16
	population_limit          uint16
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
	garrison_allotments map[*RisqUnit]bool
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
	// Zones whose cosmetic terrain_override should clear at the start of cleanupDeleted's NEXT call,
	// so a destroyed building's override is still visible for the turn following its death
	pending_terrain_clears []*RisqZone
	// Techs finishing production this tick; applied after this tick's health deltas so a tech's
	// combat bonus never affects the same tick's combat, only the next one
	pending_tech_completions []techCompletion
	completed_gatherables    []*RisqBuilding
	regions                  []*RisqRegion
	// owned by this game only, never the shared global source, so concurrent AI goroutines can't race it
	rng           *rand.Rand
	ai_goroutines sync.WaitGroup
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
	r.endTurn()
	r.checkWinCondition()
	if !r.game.GameEnded() {
		r.startNextTurn()
	}
}

func (r *GameRisq) endTurn() {
	r.recalculateOwnership()
	r.resolvePendingMercenaries()
	r.recalculateVision()
	r.refreshScores()
	r.updateEliminated()
	r.finalizeTurnReports()
	r.logStateHash("end")
}

func (r *GameRisq) startNextTurn() {
	r.recordEconomyStamina()
	r.turn_number++
	for _, player := range r.players {
		player.orders_submitted = false
	}
	for o := range r.allOrderables() {
		o.refreshStamina()
	}
	r.giving_orders = true
	for _, player := range r.players {
		player.player.AddUpdate(&game.UpdateMessage{Kind: "start-turn", Content: gin.H{
			"game": r.toFrontendFor(player.player.Player_id, player.player.GetClientId(), false),
		}})
	}
	r.game.AddViewerUpdate(&game.UpdateMessage{Kind: "start-turn", Content: gin.H{
		"game": r.ToFrontend(0, true),
	}})
}

func (r *GameRisq) updateEliminated() {
	for _, player := range r.players {
		if player.eliminated {
			continue
		}
		if len(player.units) == 0 && len(player.buildings) == 0 {
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

// Keyed by player id since every AI player shares client id 0
func (r *GameRisq) toFrontendFor(player_id int, client_id uint64, is_viewer bool) gin.H {
	game := gin.H{
		"board_size":       r.board_size,
		"population_limit": r.population_limit,
		"turn_number":      r.turn_number,
		"giving_orders":    r.giving_orders,
	}
	if r.game != nil {
		game["game_base"] = r.game.ToFrontend(client_id, is_viewer)
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
	regions := []gin.H{}
	for _, region := range r.regions {
		if reg := region.toFrontend(r, player_id); reg != nil {
			regions = append(regions, reg)
		}
	}
	game["regions"] = regions
	return game
}
