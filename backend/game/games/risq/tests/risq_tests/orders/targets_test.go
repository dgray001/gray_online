package orders

import (
	"fmt"
	"strings"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

func TestInvalidOrderTargets(t *testing.T) {
	cases := []struct {
		typ     defs.OrderType
		target  int64
		message string
	}{
		{defs.OrderType_UnitMoveSpace, harness.SpaceKey(9, 0), "Invalid space target inverted from"},
		{defs.OrderType_UnitMoveZone, harness.ZoneKey(9, 0, 0, 0), "Invalid space target inverted from zone key"},
		{defs.OrderType_UnitMoveZone, harness.ZoneKey(0, 0, 9, 0), "Invalid zone target inverted from zone key"},
		{defs.OrderType_UnitGather, harness.ZoneKey(0, 0, 1, 0), "No resource in target zone"},
		{defs.OrderType_UnitGather, harness.ZoneKey(9, 0, 0, 0), "Invalid space target inverted from zone key"},
		{defs.OrderType_UnitGather, harness.ZoneKey(0, 0, 9, 0), "Invalid zone target inverted from zone key"},
		{defs.OrderType_UnitBuild, harness.BuildKey(999, 1, 0, 0, 0), "Invalid or unbuildable building id:"},
		{defs.OrderType_UnitBuild, harness.BuildKey(2, 9, 0, 0, 0), "Invalid space or zone target inverted from build key"},
		{defs.OrderType_UnitBuild, harness.BuildKey(2, 0, 0, 1, -1), "Target zone is already occupied"},
		{defs.OrderType_UnitBuild, harness.BuildKey(2, 0, 0, 0, 0), "Target zone is already occupied"},
		{defs.OrderType_UnitBuild, harness.BuildKey(3, 1, 0, 0, 0), "Required tech id 1 not researched"},
		{defs.OrderType_UnitRepair, 999999, "Invalid building target id"},
		{defs.OrderType_UnitRenew, 999999, "Invalid building target id"},
		{defs.OrderType_UnitAttackUnit, 999999, "Invalid unit target id"},
		{defs.OrderType_UnitAttackBuilding, 999999, "Invalid building target id"},
		{defs.OrderType_UnitAttackZone, harness.ZoneKey(9, 0, 0, 0), "Invalid space target inverted from zone key"},
		{defs.OrderType_UnitAttackZone, harness.ZoneKey(0, 0, 9, 0), "Invalid zone target inverted from zone key"},
		{defs.OrderType_UnitAttackSpace, harness.SpaceKey(9, 0), "Invalid space target inverted from"},
		{defs.OrderType_UnitGarrison, 999999, "Invalid building target id"},
		{defs.OrderType_BuildingCreate, 999999, "Invalid or unsupported unit id for production:"},
		{defs.OrderType_BuildingResearch, 999999, "Invalid or unsupported tech id:"},
		{defs.OrderType_BuildingAttackUnit, 999999, "Invalid unit target id"},
		{defs.OrderType_BuildingAttackBuilding, 999999, "Invalid building target id"},
		{defs.OrderType_CancelOrder, 999999, "No active order with id"},
		{defs.OrderType_CancelFoundation, harness.ZoneKey(9, 0, 0, 0), "Invalid zone target inverted from zone key"},
		{defs.OrderType_CancelFoundation, harness.ZoneKey(1, 0, 0, 0), "No planned foundation at this zone"},
		{defs.OrderType_BuyMercenary, harness.BuildKey(11, 9, 0, 0, 0), "Invalid space or zone target inverted from mercenary key"},
		{defs.OrderType_BuyMercenary, harness.BuildKey(999, 0, 0, 0, 0), "Invalid or unsupported unit id:"},
		{defs.OrderType_BuyMercenary, harness.BuildKey(13, 0, 0, 0, 0), "not an available mercenary"},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%d/%d", c.typ, c.target), func(t *testing.T) {
			g := orderGame(t, richBank, "")
			p := g.Human(0)
			order := validOrder(g, p, c.typ)
			order.Player_id, order.Target_id = g.PlayerID(p), c.target
			message := rejectAction(g, p, "submit-orders", gin.H{"orders": []defs.OrderFromFrontend{order}}, "")
			if !strings.Contains(message, c.message) {
				t.Fatalf("failure %q, want %q", message, c.message)
			}
		})
	}
}
