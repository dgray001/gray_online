package invariants

import (
	"fmt"
	"slices"
	"sync/atomic"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/config"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

func TestDeterminismAndInvariants(t *testing.T) {
	for name, scenario := range map[string]scenario{"economy": economyScenario(), "gathering": gatheringScenario(), "construction": constructionScenario(), "combat": combatScenario(), "eviction": evictionScenario()} {
		t.Run(name, func(t *testing.T) {
			permutations := [][]int{{0, 1}, {1, 0}}
			if scenario.players == 3 {
				permutations = [][]int{{0, 1, 2}, {0, 2, 1}, {1, 0, 2}, {1, 2, 0}, {2, 0, 1}, {2, 1, 0}}
			}
			baseline := runScenario(t, scenario, permutations[0], true)
			configShared.Store(true)
			t.Cleanup(func() { configShared.Store(false) })
			for repeat := range 3 {
				for i, permutation := range permutations {
					if repeat == 0 && i == 0 {
						continue
					}
					t.Run(fmt.Sprintf("run%d/%v", repeat, permutation), func(t *testing.T) {
						t.Parallel()
						compareTrajectory(t, baseline, runScenario(t, scenario, permutation, false))
					})
				}
			}
		})
	}
}

type scenario struct {
	players     int
	turns       int
	capLossTurn int
	setup       func(*testing.T) *harness.Game
	orders      func(*harness.Game, int, int) []defs.OrderFromFrontend
	verify      func(*harness.Game, int)
}

func noConfig() {}

// Set while parallel runs share the config the scenario's first game applied
var configShared atomic.Bool

func invariantGame(t *testing.T, players int, spaces, extra string, configure func()) *harness.Game {
	t.Helper()
	if !configShared.Load() {
		doc := fmt.Sprintf(`{"board_size":1,"players":%d,"starting_bank":{"food":500,"wood":500,"stone":500,"gold":500},"spaces":[%s]%s}`, players, spaces, extra)
		fakeboard.UseConfig(t, testconfig.Dir(), nil, map[string]string{"invariants": doc})
		configure()
		allSeeingConfig()
	}
	g := harness.NewGame(t, "custom:invariants", 37, players)
	g.Lite = true
	return g
}

func runScenario(t *testing.T, scenario scenario, permutation []int, full bool) []string {
	t.Helper()
	g := scenario.setup(t)
	hashes := make([]string, 0, scenario.turns)
	for turn := 0; turn < scenario.turns; turn++ {
		batches := make([][]defs.OrderFromFrontend, scenario.players)
		for slot := range batches {
			batches[slot] = scenario.orders(g, turn, slot)
		}
		before := g.State(g.Human(0)).TurnNumber
		for _, slot := range permutation {
			g.Submit(g.Human(slot), batches[slot]...)
		}
		if after := g.State(g.Human(0)).TurnNumber; after != before+1 {
			t.Fatalf("turn advanced from %d to %d", before, after)
		}
		scenario.verify(g, turn)
		hashes = append(hashes, fmt.Sprintf("%016x", g.Risq.StateHash()))
	}
	if full {
		checkState(t, g, scenario.players, scenario.capLossTurn == 0 || scenario.turns < scenario.capLossTurn)
	}
	return hashes
}

func compareTrajectory(t *testing.T, want, got []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("trajectory length %d, want %d", len(got), len(want))
	}
	for turn := range want {
		if got[turn] == want[turn] {
			continue
		}
		i := 0
		for i < min(len(got[turn]), len(want[turn])) && got[turn][i] == want[turn][i] {
			i++
		}
		start := max(0, i-80)
		t.Fatalf("state %d differs at byte %d:\n got ...%s\nwant ...%s", turn, i, got[turn][start:min(len(got[turn]), i+160)], want[turn][start:min(len(want[turn]), i+160)])
	}
}

func unitIDs(g *harness.Game, slot int, unitID uint32) []uint64 {
	var ids []uint64
	for _, unit := range g.Self(g.Human(slot)).Units {
		if unit.UnitID == unitID {
			ids = append(ids, unit.InternalID)
		}
	}
	slices.Sort(ids)
	return ids
}

func buildingID(g *harness.Game, slot int, buildingID uint32) uint64 {
	for _, building := range g.Self(g.Human(slot)).Buildings {
		if building.BuildingID == buildingID {
			return building.InternalID
		}
	}
	g.T.Fatalf("player %d missing building type %d", slot, buildingID)
	return 0
}

func allSeeingConfig() {
	vision := defs.RisqVision{Space: 4, Edge_adjacent: 4, Adjacent: 4, Edge_opposite: 4, Secondary: 4}
	for id, config := range defs.UnitConfigs {
		config.Vision = vision
		defs.UnitConfigs[id] = config
	}
	for id, config := range defs.BuildingConfigs {
		config.Vision = vision
		defs.BuildingConfigs[id] = config
	}
}
