package simoutcome

import "github.com/dgray001/gray_online/game/games/risq"

// How a game ended from one player's side; every game is exactly one of these for each player
type Outcome string

const (
	OutcomeDefeat   Outcome = "defeat"
	OutcomeCrush    Outcome = "crush"
	OutcomeWinning  Outcome = "winning"
	OutcomeAhead    Outcome = "ahead"
	OutcomeBehind   Outcome = "behind"
	OutcomeLosing   Outcome = "losing"
	OutcomeCrushed  Outcome = "crushed"
	OutcomeDefeated Outcome = "defeated"
)

// One letter each, for a game's whole path on one line
var Letters = map[Outcome]string{
	OutcomeDefeat: "D", OutcomeCrush: "C", OutcomeWinning: "W", OutcomeAhead: "A",
	OutcomeBehind: "B", OutcomeLosing: "L", OutcomeCrushed: "X", OutcomeDefeated: "Z",
}

// Best to worst, the order the reports list them in
var Order = []Outcome{OutcomeDefeat, OutcomeCrush, OutcomeWinning, OutcomeAhead, OutcomeBehind, OutcomeLosing, OutcomeCrushed, OutcomeDefeated}

// At least factor times the other's units, and some units at all
func outnumbers(a risq.PlayerStanding, b risq.PlayerStanding, factor int) bool {
	return a.Units > 0 && a.Units >= factor*b.Units
}

// At least twice the units, more score and more land
func winning(a risq.PlayerStanding, b risq.PlayerStanding) bool {
	return outnumbers(a, b, 2) && a.Score > b.Score && a.Land > b.Land
}

// More score; an exact tie goes to the lower player id, so there is no tie outcome
func ahead(a risq.PlayerStanding, b risq.PlayerStanding) bool {
	return a.Score > b.Score || (a.Score == b.Score && a.PlayerId < b.PlayerId)
}

// Each seat's outcome against the other seat; only two-player games are judged
func Outcomes(standings []risq.PlayerStanding) []Outcome {
	if len(standings) != 2 {
		return nil
	}
	return []Outcome{Classify(standings[0], standings[1]), Classify(standings[1], standings[0])}
}

// The outcomes the engine's end-of-game results give
func FinalOutcomes(players []risq.PlayerResult) []Outcome {
	standings := make([]risq.PlayerStanding, len(players))
	for i, p := range players {
		standings[i] = risq.PlayerStanding{PlayerId: p.PlayerId, Units: p.Units, Score: p.Score, Land: p.Land, Eliminated: p.Eliminated}
	}
	return Outcomes(standings)
}

// The first that holds, in order: the engine eliminated one side only, a 10x unit lead, the winning test, then score
func Classify(self risq.PlayerStanding, other risq.PlayerStanding) Outcome {
	switch {
	case other.Eliminated && !self.Eliminated:
		return OutcomeDefeat
	case self.Eliminated && !other.Eliminated:
		return OutcomeDefeated
	case outnumbers(self, other, 10):
		return OutcomeCrush
	case outnumbers(other, self, 10):
		return OutcomeCrushed
	case winning(self, other):
		return OutcomeWinning
	case winning(other, self):
		return OutcomeLosing
	case ahead(self, other):
		return OutcomeAhead
	}
	return OutcomeBehind
}
