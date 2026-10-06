package orders

import (
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

func TestGatherPointAttacksVisibleEnemyObjects(t *testing.T) {
	for _, object := range []uint8{1, 2} {
		t.Run(fmt.Sprint(object), func(t *testing.T) {
			g, p := gatherCombatGame(t, 22, 1, 13)
			target, typ := unitIDs(g, g.Human(1), 13)[0], defs.OrderType_UnitAttackUnit
			before := g.Self(g.Human(1)).Units[0].CombatStats.Health
			if object == 2 {
				target, typ, before = buildingID(g, g.Human(1), 2), defs.OrderType_UnitAttackBuilding, g.Self(g.Human(1)).Buildings[0].CombatStats.Health
			}
			u := produceAtPoint(g, p, 22, 11, gin.H{"location_kind": 2, "location_id": harness.ZoneKey(0, 0, 1, 0), "object_type": object, "object_id": target})
			if len(u.ActiveOrders) != 1 || u.ActiveOrders[0].OrderType != uint8(typ) {
				t.Fatalf("new unit did not attack: %+v", u)
			}
			after := g.Self(g.Human(1)).Units[0].CombatStats.Health
			if object == 2 {
				after = g.Self(g.Human(1)).Buildings[0].CombatStats.Health
			}
			if after >= before {
				t.Fatalf("gather-point attack caused no damage: before %v, after %v", before, after)
			}
		})
	}
}
