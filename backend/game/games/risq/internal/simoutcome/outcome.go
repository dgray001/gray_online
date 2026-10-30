package simoutcome

import (
	"maps"
	"slices"

	"github.com/dgray001/gray_online/game/games/risq"
)

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

// Alive beats eliminated, then more score, then the lower id
func represents(a risq.PlayerStanding, b risq.PlayerStanding) bool {
	if a.Eliminated != b.Eliminated {
		return !a.Eliminated
	}
	return ahead(a, b)
}

// Each seat's outcome for its side (sides[i] is standings[i]'s side), comparing the two sides' best players; judged only with exactly two sides
func Outcomes(standings []risq.PlayerStanding, sides []int) []Outcome {
	if len(sides) != len(standings) {
		return nil
	}
	best := map[int]risq.PlayerStanding{}
	for i, s := range standings {
		if current, found := best[sides[i]]; !found || represents(s, current) {
			best[sides[i]] = s
		}
	}
	if len(best) != 2 {
		return nil
	}
	ids := slices.Sorted(maps.Keys(best))
	verdict := map[int]Outcome{ids[0]: Classify(best[ids[0]], best[ids[1]]), ids[1]: Classify(best[ids[1]], best[ids[0]])}
	outcomes := make([]Outcome, len(sides))
	for i, side := range sides {
		outcomes[i] = verdict[side]
	}
	return outcomes
}

// The outcomes the engine's end-of-game results give
func FinalOutcomes(players []risq.PlayerResult, sides []int) []Outcome {
	standings := make([]risq.PlayerStanding, len(players))
	for i, p := range players {
		standings[i] = risq.PlayerStanding{PlayerId: p.PlayerId, Units: p.Units, Score: p.Score, Land: p.Land, Eliminated: p.Eliminated}
	}
	return Outcomes(standings, sides)
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
