package risq

import (
	"math/rand"
	"sort"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/game_utils"
)

type TTKResult struct {
	Turns    int
	Ticks    int
	Possible bool
}

// One space with two units/a building placed in its center zone, for isolated 1v1 combat sims.
func newTTKGame() (*GameRisq, *RisqPlayer, *RisqPlayer) {
	base := game.CreateBaseGame(0, game.GameType_RISQ, nil)
	game.CreatePlayer(1, "attacker", base)
	game.CreatePlayer(2, "target", base)
	base.Players[1].Player_id = 0
	base.Players[2].Player_id = 1
	rng := rand.New(rand.NewSource(0))
	attacker_player := createRisqPlayer(base.Players[1], 100, "red", rng)
	target_player := createRisqPlayer(base.Players[2], 100, "blue", rng)
	r := &GameRisq{
		players:   []*RisqPlayer{attacker_player, target_player},
		units:     make(map[uint64]*RisqUnit),
		buildings: make(map[uint64]*RisqBuilding),
		spaces:    [][]*RisqSpace{{createRisqSpace(0, 0, defaultTerrainId)}},
	}
	return r, attacker_player, target_player
}

func markResearched(player *RisqPlayer, tech_ids []uint32) {
	for _, id := range tech_ids {
		player.researched_techs[id] = true
	}
}

// Drives real per-turn stamina regen and calls attack (a real unitAttack call) once per tick,
// exactly like resolveActiveOrders' tick loop but for one attacker only.
func runTTKLoop(attacker *RisqUnit, healthOf func() float64, attack func(stamina_cost int)) TTKResult {
	turns := 0
	ticks := 0
	for healthOf() > 0 && turns < 100000 {
		turns++
		attacker.refreshStamina()
		for attacker.current_stamina > 0 && healthOf() > 0 {
			cost := min(attacker.current_stamina, unitTickStaminaCost)
			before := healthOf()
			attack(cost)
			attacker.current_stamina -= cost
			ticks++
			if healthOf() == before {
				return TTKResult{}
			}
		}
	}
	if healthOf() > 0 {
		return TTKResult{}
	}
	return TTKResult{Turns: turns, Ticks: ticks, Possible: true}
}

func ComputeUnitVsUnitTTK(attacker_unit_id uint32, target_unit_id uint32) TTKResult {
	return ComputeUnitVsUnitTTKWithTechs(attacker_unit_id, target_unit_id, nil, nil)
}

func ComputeUnitVsUnitTTKWithTechs(attacker_unit_id uint32, target_unit_id uint32, attacker_tech_ids []uint32, target_tech_ids []uint32) TTKResult {
	if _, ok := unitConfigs[attacker_unit_id]; !ok {
		return TTKResult{}
	}
	if _, ok := unitConfigs[target_unit_id]; !ok {
		return TTKResult{}
	}
	r, attacker_player, target_player := newTTKGame()
	markResearched(attacker_player, attacker_tech_ids)
	markResearched(target_player, target_tech_ids)
	attacker := createRisqUnit(1, attacker_unit_id, attacker_player)
	target := createRisqUnit(2, target_unit_id, target_player)
	space := r.spaces[0][0]
	space.setUnit(&game_utils.Coordinate2D{}, attacker)
	space.setUnit(&game_utils.Coordinate2D{}, target)
	return runTTKLoop(attacker, func() float64 { return target.cs.health }, func(cost int) {
		attacker.intent.intent_cost = cost
		r.unitAttack(attacker, target)
		target.resolveHealthDelta(r)
	})
}

func ComputeUnitVsBuildingTTK(attacker_unit_id uint32, target_building_id uint32) TTKResult {
	return ComputeUnitVsBuildingTTKWithTechs(attacker_unit_id, target_building_id, nil)
}

func ComputeUnitVsBuildingTTKWithTechs(attacker_unit_id uint32, target_building_id uint32, attacker_tech_ids []uint32) TTKResult {
	if _, ok := unitConfigs[attacker_unit_id]; !ok {
		return TTKResult{}
	}
	if _, ok := buildingConfigs[target_building_id]; !ok {
		return TTKResult{}
	}
	r, attacker_player, target_player := newTTKGame()
	markResearched(attacker_player, attacker_tech_ids)
	attacker := createRisqUnit(1, attacker_unit_id, attacker_player)
	target := createRisqBuilding(1, target_building_id, target_player.player.Player_id)
	space := r.spaces[0][0]
	space.setUnit(&game_utils.Coordinate2D{}, attacker)
	space.setBuilding(&game_utils.Coordinate2D{}, target)
	target_player.buildings[target.internal_id] = target
	return runTTKLoop(attacker, func() float64 { return target.cs.health }, func(cost int) {
		attacker.intent.intent_cost = cost
		r.unitAttack(attacker, target)
		target.resolveHealthDelta(r)
	})
}

func AllUnitIDs() []uint32 {
	ids := make([]uint32, 0, len(unitConfigs))
	for id := range unitConfigs {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func AllBuildingIDs() []uint32 {
	ids := make([]uint32, 0, len(buildingConfigs))
	for id := range buildingConfigs {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func UnitName(id uint32) string {
	return unitConfigs[id].display_name
}

func BuildingName(id uint32) string {
	return buildingConfigs[id].display_name
}
