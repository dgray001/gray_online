package shape

import (
	"fmt"
	"testing"
)

func TestEntityCopiesAgreeAcrossSnapshot(t *testing.T) {
	for _, level := range []uint8{2, 3, 4} {
		t.Run(fmt.Sprint(level), func(t *testing.T) {
			g := shapeGame(t, level)
			queueWork(g)
			for _, human := range []int{g.Human(0), g.Human(1)} {
				state := snapshot(g, human, false)
				target := space(t, state, 0, 0)
				checkWorldCopies(t, state, target)
			}
		})
	}
}
