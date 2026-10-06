package invariants

import (
	"slices"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

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
