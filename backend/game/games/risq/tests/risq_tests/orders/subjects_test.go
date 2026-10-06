package orders

import (
	"slices"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

const richBank = `{"food":1000,"wood":1000,"stone":1000,"gold":1000}`

func unitIDs(g *harness.Game, p int, kind uint32) []uint64 {
	var ids []uint64
	for _, u := range g.Self(p).Units {
		if u.UnitID == kind {
			ids = append(ids, u.InternalID)
		}
	}
	slices.Sort(ids)
	return ids
}
