package orders

import (
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

func TestOnlyEconomicUnitsMayWork(t *testing.T) {
	for typ, message := range map[defs.OrderType]string{
		defs.OrderType_UnitGather: "Only economic units can gather", defs.OrderType_UnitBuild: "Only economic units can build",
		defs.OrderType_UnitRepair: "Only economic units can repair", defs.OrderType_UnitRenew: "Only economic units can renew",
	} {
		t.Run(fmt.Sprint(typ), func(t *testing.T) {
			g := orderGame(t, richBank, "")
			p := g.Human(0)
			order := validOrder(g, p, typ)
			order.Subjects = unitIDs(g, p, 11)
			g.SubmitRejected(p, order, message)
		})
	}
}
