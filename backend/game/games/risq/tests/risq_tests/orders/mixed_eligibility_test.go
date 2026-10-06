package orders

import (
	"fmt"
	"strings"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

func TestMixedEligibleSubjectsRejectWholeOrder(t *testing.T) {
	for typ, message := range map[defs.OrderType]string{
		defs.OrderType_UnitGather:             "Only economic units can gather",
		defs.OrderType_UnitBuild:              "Only economic units can build",
		defs.OrderType_UnitRepair:             "Only economic units can repair",
		defs.OrderType_UnitRenew:              "Only economic units can renew",
		defs.OrderType_BuildingCreate:         "Building id 11 cannot produce unit id 1",
		defs.OrderType_BuildingResearch:       "Building id 11 cannot research tech id 1",
		defs.OrderType_UnitAttackSpace:        "Unit id %d cannot attack",
		defs.OrderType_UnitAttackZone:         "Unit id %d cannot attack",
		defs.OrderType_UnitAttackUnit:         "Unit id %d cannot attack",
		defs.OrderType_UnitAttackBuilding:     "Unit id %d cannot attack",
		defs.OrderType_BuildingAttackUnit:     "Building id %d cannot attack",
		defs.OrderType_BuildingAttackBuilding: "Building id %d cannot attack",
	} {
		t.Run(fmt.Sprint(typ), func(t *testing.T) {
			useOrderConfig(t, richBank, "")
			if typ.IsUnitOrder() && typ.IsAttackOrder() {
				config := defs.UnitConfigs[11]
				config.Attack_type = defs.AttackType_NONE
				defs.UnitConfigs[11] = config
			}
			g := harness.NewGame(t, "custom:orders", 1, 2)
			p := g.Human(0)
			order := validOrder(g, p, typ)
			eligible, ineligible := order.Subjects[0], unitIDs(g, p, 11)[0]
			if typ.IsBuildingOrder() {
				ineligible = buildingID(g, p, 11)
			}
			if strings.Contains(message, "%d") {
				message = fmt.Sprintf(message, ineligible)
			}
			order.Player_id = g.PlayerID(p)
			for _, subjects := range [][]uint64{{eligible, ineligible}, {ineligible, eligible}} {
				order.Subjects = subjects
				rejectAction(g, p, "submit-orders", gin.H{"orders": []defs.OrderFromFrontend{order}}, message)
			}
		})
	}
}
