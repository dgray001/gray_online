package risq

type PlayerResult struct {
	PlayerId      int
	Nickname      string
	Score         uint
	Eliminated    bool
	Units         int
	Buildings     int
	Kills         uint
	Razes         uint
	UnitsLost     uint
	BuildingsLost uint
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
		players = append(players, PlayerResult{
			PlayerId:      p.player.Player_id,
			Nickname:      p.player.GetNickname(),
			Score:         p.score,
			Eliminated:    p.eliminated,
			Units:         len(p.units),
			Buildings:     len(p.buildings),
			Kills:         p.kills,
			Razes:         p.razes,
			UnitsLost:     p.units_lost,
			BuildingsLost: p.buildings_lost,
		})
	}
	return GameResult{TurnNumber: r.turn_number, Players: players}
}
