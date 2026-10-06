package risq_mapgen_tests

import (
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

// sim_wide's start areas overlap for most player counts, which once made mirrored start resources fail
func TestSimWideStartsMirrorForEveryCombination(t *testing.T) {
	for _, players := range []int{3, 4, 6, 8, 12} {
		for seed := int64(1); seed <= 3; seed++ {
			t.Run(fmt.Sprintf("p%d/s%d", players, seed), func(t *testing.T) {
				board, err := fakeboard.Generate("script:sim_wide", players, seed)
				if err != nil {
					t.Fatal(err)
				}
				checkBoardInvariants(t, board, players)
			})
		}
	}
}
