package orders

import (
	"cmp"
	"slices"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func normalized(p harness.PlayerState) harness.PlayerState {
	slices.SortFunc(p.Units, func(a, b harness.Unit) int { return cmp.Compare(a.InternalID, b.InternalID) })
	slices.SortFunc(p.Buildings, func(a, b harness.Building) int { return cmp.Compare(a.InternalID, b.InternalID) })
	return p
}
