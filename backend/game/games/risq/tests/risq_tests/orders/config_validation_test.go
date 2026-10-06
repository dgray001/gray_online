package orders

import (
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestConfigRestrictionsAreEnforced(t *testing.T) {
	for _, kind := range []string{"unit tech", "research tech", "build list", "unarmed unit"} {
		t.Run(kind, func(t *testing.T) {
			useOrderConfig(t, richBank, "")
			unit, tech := defs.UnitConfigs[1], defs.TechConfigs[1]
			switch kind {
			case "unit tech":
				unit.Required_tech_id = 3
			case "research tech":
				tech.Required_tech_id = 3
			case "build list":
				unit.Builds = nil
			case "unarmed unit":
				unit.Attack_type = defs.AttackType_NONE
			}
			defs.UnitConfigs[1], defs.TechConfigs[1] = unit, tech
			g := harness.NewGame(t, "custom:orders", 1, 2)
			p := g.Human(0)
			order := harness.OrderProduce([]uint64{buildingID(g, p, 1)}, 1)
			message := "Required tech id 3 not researched for unit id 1"
			switch kind {
			case "research tech":
				order.Order_type, message = uint8(defs.OrderType_BuildingResearch), "Required tech id 3 not researched for tech id 1"
			case "build list":
				order, message = harness.OrderBuild(unitIDs(g, p, 1)[:1], 2, 1, 0, 0, 0), "Unit id 1 cannot build building id 2"
			case "unarmed unit":
				id := unitIDs(g, p, 1)[0]
				order, message = harness.OrderAttackUnit([]uint64{id}, unitIDs(g, g.Human(1), 1)[0]), fmt.Sprintf("Unit id %d cannot attack", id)
			}
			g.SubmitRejected(p, order, message)
		})
	}
}
