package risq

import (
	"fmt"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

type GatheredResult struct {
	Food  float64
	Wood  float64
	Stone float64
	Gold  float64
}

type PlayerResult struct {
	PlayerId        int
	Nickname        string
	Score           uint
	Eliminated      bool
	Units           int
	Buildings       int
	Kills           uint
	Razes           uint
	UnitsLost       uint
	BuildingsLost   uint
	TechsResearched int
	Land            int
	Gathered        GatheredResult
	Economy         EconomyResult
}

// Idle percentages are the share of available stamina left unspent; Early covers the first earlyGameTurns turns
type EconomyResult struct {
	VillagersLost             uint
	VillagerIdlePct           float64
	VillagerIdlePctEarly      float64
	VillageCenterIdlePct      float64
	VillageCenterIdlePctEarly float64
}

type GameResult struct {
	TurnNumber uint16
	Players    []PlayerResult
	Metrics    GameMetrics
}

func (r *GameRisq) TurnNumber() uint16 {
	return r.turn_number
}

func (r *GameRisq) Results() GameResult {
	players := make([]PlayerResult, 0, len(r.players))
	for _, p := range r.players {
		techs_researched := 0
		for _, researched := range p.researched_techs {
			if researched {
				techs_researched++
			}
		}
		gathered := p.resources.LifetimeGathered()
		players = append(players, PlayerResult{
			PlayerId:        p.player.Player_id,
			Nickname:        p.player.GetNickname(),
			Score:           p.score,
			Eliminated:      p.eliminated,
			Units:           len(p.units),
			Buildings:       len(p.buildings),
			Kills:           p.kills,
			Razes:           p.razes,
			UnitsLost:       p.units_lost,
			BuildingsLost:   p.buildings_lost,
			TechsResearched: techs_researched,
			Land:            r.ownedCount(p.player.Player_id),
			Gathered:        GatheredResult{Food: gathered.Food, Wood: gathered.Wood, Stone: gathered.Stone, Gold: gathered.Gold},
			Economy: EconomyResult{
				VillagersLost:             p.economy.villagers_lost,
				VillagerIdlePct:           p.economy.villagers.idlePct(),
				VillagerIdlePctEarly:      p.economy.villagers_early.idlePct(),
				VillageCenterIdlePct:      p.economy.village_center.idlePct(),
				VillageCenterIdlePctEarly: p.economy.village_center_early.idlePct(),
			},
		})
	}
	return GameResult{TurnNumber: r.turn_number, Players: players, Metrics: r.Metrics()}
}

// Compact per-player state used by the sim tool to print timelines
type PlayerSnapshot struct {
	Turn      uint16
	Units     map[uint32]int
	Buildings map[uint32]int
	Food      float64
	Wood      float64
	Stone     float64
	Gold      float64
	Score     uint
	// space "x,y" -> military units there (garrisoned units excluded)
	Army map[string]int
	Home string
}

func (r *GameRisq) Snapshot() []PlayerSnapshot {
	out := make([]PlayerSnapshot, 0, len(r.players))
	for _, p := range r.players {
		s := PlayerSnapshot{Turn: r.turn_number, Army: map[string]int{}, Units: map[uint32]int{}, Buildings: map[uint32]int{}, Score: p.score,
			Food: p.resources.food, Wood: p.resources.wood, Stone: p.resources.stone, Gold: p.resources.gold}
		for _, u := range p.units {
			s.Units[u.unit_id]++
			if u.unitType() == defs.UnitType_INFANTRY && u.zone != nil && u.garrisoned_in == nil {
				s.Army[fmt.Sprintf("%d,%d", u.zone.space.coordinate.X, u.zone.space.coordinate.Y)]++
			}
		}
		for _, b := range p.buildings {
			if !b.underConstruction() {
				s.Buildings[b.building_id]++
			}
			if b.building_id == 1 && b.zone != nil {
				s.Home = fmt.Sprintf("%d,%d", b.zone.space.coordinate.X, b.zone.space.coordinate.Y)
			}
		}
		out = append(out, s)
	}
	return out
}
