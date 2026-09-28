package risq

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
	return GameResult{TurnNumber: r.turn_number, Players: players}
}
