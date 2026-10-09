package risq_mapgen_tests

import (
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

func TestTriangleStartingCornerAndDirectionVary(t *testing.T) {
	useStartLayouts(t)
	patterns := map[string]bool{}
	for seed := int64(1); seed <= 64; seed++ {
		board, err := fakeboard.Generate("script:triangle", 4, seed)
		if err != nil {
			t.Fatal(err)
		}
		_, indices := triangleStartIndices(t, board, 4, 1)
		patterns[fmt.Sprint(indices)] = true
	}
	if len(patterns) != 6 {
		t.Errorf("triangle only uses %d of 6 corner/direction patterns: %v", len(patterns), patterns)
	}
}

func dumpOf(t *testing.T, name string, players int, seed int64) string {
	t.Helper()
	board, err := fakeboard.Generate("script:"+name, players, seed)
	if err != nil {
		t.Fatal(err)
	}
	return board.Dump()
}

func TestSameSeedGivesIdenticalBoard(t *testing.T) {
	for _, name := range sweptScripts {
		if dumpOf(t, name, 4, 11) != dumpOf(t, name, 4, 11) {
			t.Errorf("%s differs between two runs of the same seed", name)
		}
	}
}

func TestDifferentSeedsGiveDifferentBoards(t *testing.T) {
	for _, name := range sweptScripts {
		dumps := map[string]bool{}
		for seed := int64(20); seed < 24; seed++ {
			dumps[dumpOf(t, name, 2, seed)] = true
		}
		if len(dumps) < 2 {
			t.Errorf("%s ignores its seed: %d distinct boards from 4 seeds", name, len(dumps))
		}
	}
}
