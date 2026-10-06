package simoutcome_test

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq"
	. "github.com/dgray001/gray_online/game/games/risq/internal/simoutcome"
)

func player(id int, units int, score uint, land int, eliminated bool) risq.PlayerStanding {
	return risq.PlayerStanding{PlayerId: id, Units: units, Score: score, Land: land, Eliminated: eliminated}
}

var mirror = map[Outcome]Outcome{
	OutcomeDefeat: OutcomeDefeated, OutcomeDefeated: OutcomeDefeat, OutcomeCrush: OutcomeCrushed, OutcomeCrushed: OutcomeCrush,
	OutcomeWinning: OutcomeLosing, OutcomeLosing: OutcomeWinning, OutcomeAhead: OutcomeBehind, OutcomeBehind: OutcomeAhead,
}

func TestClassifyEachOutcome(t *testing.T) {
	cases := map[string]struct {
		self, other risq.PlayerStanding
		want        Outcome
	}{
		"engine eliminated the other":     {player(0, 5, 1, 1, false), player(1, 90, 9, 9, true), OutcomeDefeat},
		"engine eliminated us":            {player(0, 90, 9, 9, true), player(1, 5, 1, 1, false), OutcomeDefeated},
		"ten times the units":             {player(0, 50, 1, 1, false), player(1, 5, 9, 9, false), OutcomeCrush},
		"a tenth of the units":            {player(0, 5, 9, 9, false), player(1, 50, 1, 1, false), OutcomeCrushed},
		"no units against some":           {player(0, 0, 9, 9, false), player(1, 1, 1, 1, false), OutcomeCrushed},
		"twice the units, score, land":    {player(0, 20, 5, 5, false), player(1, 10, 4, 4, false), OutcomeWinning},
		"outnumbered twice over":          {player(0, 10, 4, 4, false), player(1, 20, 5, 5, false), OutcomeLosing},
		"twice the units but less land":   {player(0, 20, 5, 3, false), player(1, 10, 4, 4, false), OutcomeAhead},
		"more score, similar units":       {player(0, 10, 5, 1, false), player(1, 10, 4, 9, false), OutcomeAhead},
		"less score":                      {player(0, 10, 4, 9, false), player(1, 10, 5, 1, false), OutcomeBehind},
		"tied score, lower id is ahead":   {player(0, 10, 5, 1, false), player(1, 10, 5, 1, false), OutcomeAhead},
		"tied score, higher id is behind": {player(1, 10, 5, 1, false), player(0, 10, 5, 1, false), OutcomeBehind},
		"both eliminated falls to units":  {player(0, 30, 5, 5, true), player(1, 3, 4, 4, true), OutcomeCrush},
		"nobody has units":                {player(0, 0, 5, 1, false), player(1, 0, 4, 1, false), OutcomeAhead},
	}
	for name, c := range cases {
		if got := Classify(c.self, c.other); got != c.want {
			t.Errorf("%s: %s, want %s", name, got, c.want)
		}
	}
}

// Every pairing is some outcome for each side and the two sides' outcomes are mirrors of each other, so there is no ninth bucket
func TestClassifyIsExhaustiveAndMirrored(t *testing.T) {
	seen := map[Outcome]bool{}
	for _, units := range [][2]int{{0, 0}, {0, 3}, {3, 0}, {3, 3}, {6, 3}, {3, 6}, {5, 3}, {3, 5}, {30, 3}, {3, 30}, {29, 3}, {30, 0}, {0, 30}} {
		for _, score := range [][2]uint{{1, 1}, {2, 1}, {1, 2}} {
			for _, land := range [][2]int{{1, 1}, {2, 1}, {1, 2}} {
				for flags := 0; flags < 4; flags++ {
					a := player(0, units[0], score[0], land[0], flags&1 != 0)
					b := player(1, units[1], score[1], land[1], flags&2 != 0)
					got, other := Classify(a, b), Classify(b, a)
					if mirror[got] == "" || mirror[got] != other {
						t.Fatalf("%+v vs %+v: %s and %s are not mirrors", a, b, got, other)
					}
					seen[got] = true
				}
			}
		}
	}
	for _, o := range Order {
		if !seen[o] {
			t.Errorf("outcome %s never occurs in the grid", o)
		}
	}
}
