package orders

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

func TestHealthyUngarrisonableBuildingPointFallsBackToMovement(t *testing.T) {
	g, p := gatherCombatGame(t, 1, 0, 13)
	target := buildingID(g, p, 2)
	u := produceAtPoint(g, p, 1, 1, gin.H{"location_kind": 2, "location_id": harness.ZoneKey(0, 0, 1, 0), "object_type": 2, "object_id": target})
	if u.Space != (harness.Coord{}) || u.Zone != (harness.Coord{X: 1}) || len(u.ActiveOrders) != 0 || u.GarrisonedIn != nil {
		t.Fatalf("friendly building fallback failed: %+v", u)
	}
	if entity(g, p, "buildings", target)["combat_stats"].(gin.H)["health"] != float64(120) {
		t.Fatal("new unit attacked friendly building")
	}
}
