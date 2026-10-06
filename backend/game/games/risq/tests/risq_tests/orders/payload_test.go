package orders

import (
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

func TestMalformedActions(t *testing.T) {
	for kind, payload := range map[string]gin.H{
		"submit-orders":         {"orders": "invalid"},
		"set-unit-behavior":     {"internal_ids": "invalid"},
		"set-building-behavior": {"auto_attack": "invalid"},
		"set-gather-point":      {"building_id": "invalid"},
	} {
		t.Run(kind, func(t *testing.T) {
			g := orderGame(t, richBank, "")
			rejectAction(g, g.Human(0), kind, payload, "")
		})
	}
}

func TestSubmissionIdentityAndAtomicValidation(t *testing.T) {
	g := orderGame(t, richBank, "")
	p := g.Human(0)
	order := harness.OrderMove(unitIDs(g, p, 1), 1, 0)
	order.Player_id = g.PlayerID(g.Human(1))
	rejectAction(g, p, "submit-orders", gin.H{"orders": []defs.OrderFromFrontend{order}}, fmt.Sprintf("Order player id %d does not match submitter %d", order.Player_id, g.PlayerID(p)))
	order.Player_id = g.PlayerID(p)
	bad := order
	bad.Subjects = []uint64{999999}
	rejectAction(g, p, "submit-orders", gin.H{"orders": []defs.OrderFromFrontend{order, bad}}, "Invalid unit subject id")
	g.Submit(p, order)
	g.EndTurn()
	if len(g.Self(p).ActiveOrders) != 0 {
		t.Fatal("valid resubmission did not finish")
	}
}
