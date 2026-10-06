package risq_mapgen_tests

import (
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

func TestScriptsGenerateForSweptPlayerCounts(t *testing.T) {
	for _, name := range sweptScripts {
		for _, players := range sweptPlayerCounts(name) {
			t.Run(fmt.Sprintf("%s/p%d", name, players), func(t *testing.T) {
				board, err := fakeboard.Generate("script:"+name, players, 1)
				if err != nil {
					t.Fatal(err)
				}
				checkBoardInvariants(t, board, players)
			})
		}
	}
}
