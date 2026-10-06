package orders

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

func TestGatherPointRepairsWhenGarrisonIsUnavailable(t *testing.T) {
	g, p := gatherCombatGame(t, 1, 0, 1)
	other, target := g.Human(1), buildingID(g, p, 2)
	g.Submit(other, harness.Order(defs.OrderType_UnitAttackBuilding, unitIDs(g, other, 1), int64(target), false))
	g.EndTurn()
	order := g.Self(other).ActiveOrders[0].InternalID
	g.Submit(other, harness.Order(defs.OrderType_CancelOrder, nil, int64(order), false))
	g.EndTurn()
	before := entity(g, p, "buildings", target)["combat_stats"].(gin.H)["health"]
	u := produceAtPoint(g, p, 1, 1, gin.H{"location_kind": 2, "location_id": harness.ZoneKey(0, 0, 1, 0), "object_type": 2, "object_id": target})
	after := entity(g, p, "buildings", target)["combat_stats"].(gin.H)["health"]
	if before.(float64) >= 120 || after.(float64) <= before.(float64) || u.GarrisonedIn != nil || g.Self(p).Resources.Wood >= 1000 {
		t.Fatalf("repair fallback failed: before %v, after %v, unit %+v", before, after, u)
	}
}
