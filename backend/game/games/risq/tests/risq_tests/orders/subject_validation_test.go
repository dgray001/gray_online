package orders

import (
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/gin-gonic/gin"
)

func TestOrderSubjectRestrictions(t *testing.T) {
	for typ := defs.OrderType_UnitMoveSpace; typ < defs.OrderType_END; typ++ {
		if typ.IsAutoSynthesized() || (!typ.IsUnitOrder() && !typ.IsBuildingOrder()) {
			continue
		}
		t.Run(fmt.Sprint(typ), func(t *testing.T) {
			g := orderGame(t, richBank, "")
			p := g.Human(0)
			order := validOrder(g, p, typ)
			order.Subjects = nil
			g.SubmitRejected(p, order, "Order must have at least one subject")
			order.Subjects = unitIDs(g, g.Human(1), 1)
			message := "Invalid unit subject id"
			if typ.IsBuildingOrder() {
				order.Subjects = []uint64{buildingID(g, g.Human(1), 1)}
				message = "Invalid building subject id"
			}
			g.SubmitRejected(p, order, message)
			owned := unitIDs(g, p, 1)[0]
			if typ.IsBuildingOrder() {
				owned = buildingID(g, p, 1)
			}
			for _, invalid := range []uint64{order.Subjects[0], 999999} {
				for _, subjects := range [][]uint64{{owned, invalid}, {invalid, owned}} {
					order.Subjects, order.Player_id = subjects, g.PlayerID(p)
					rejectAction(g, p, "submit-orders", gin.H{"orders": []defs.OrderFromFrontend{order}}, message)
				}
			}
		})
	}
}
