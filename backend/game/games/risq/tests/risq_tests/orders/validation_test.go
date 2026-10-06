package orders

import (
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

func TestEveryOrderTypeValidation(t *testing.T) {
	for typ := defs.OrderType_None; typ <= defs.OrderType_END; typ++ {
		t.Run(fmt.Sprint(typ), func(t *testing.T) {
			g := orderGame(t, richBank, `,"starting_techs":[4]`)
			p := g.Human(0)
			order := validOrder(g, p, typ)
			if typ == defs.OrderType_None || typ == defs.OrderType_END || typ.IsAutoSynthesized() {
				g.SubmitRejected(p, order, fmt.Sprintf("Invalid order type: %d", typ))
				return
			}
			g.Submit(p, order)
			if !g.Self(p).OrdersSubmitted {
				t.Fatal("valid order did not mark submitted")
			}
			g.Action(p, "unsubmit-orders", nil)
			order.Subjects = []uint64{999999}
			message := "Invalid subjects in player order"
			if typ.IsUnitOrder() {
				message = "Invalid unit subject id"
			}
			if typ.IsBuildingOrder() {
				message = "Invalid building subject id"
			}
			if typ == defs.OrderType_CancelOrder {
				message = fmt.Sprintf("Subject 999999 is not on order %d", order.Target_id)
			}
			g.SubmitRejected(p, order, message)
		})
	}
}
