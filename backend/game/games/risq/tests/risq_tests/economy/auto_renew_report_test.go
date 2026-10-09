package economy

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

func woodTurnReport(g *harness.Game, p int) gin.H {
	g.T.Helper()
	for _, player := range g.Risq.ToFrontend(uint64(p+1), false)["players"].([]gin.H) {
		if player["resources"] == nil {
			continue
		}
		for _, line := range player["turn_report"].(gin.H)["resources"].([]gin.H) {
			if line["category"] == defs.RisqResourceCategory_WOOD {
				return line
			}
		}
	}
	g.T.Fatal("wood report missing")
	return nil
}

func TestAutoRenewOrderingPaymentsAndRefundsAppearInReport(t *testing.T) {
	g := sourceGame(t, true)
	p := g.Human(0)
	g.Action(p, "set-auto-renew", gin.H{"building_id": 3, "count": 2})
	g.Action(p, "set-auto-renew", gin.H{"building_id": 3, "count": 1})
	if g.Self(p).Resources.Wood != 60 {
		t.Fatal("ordering-phase payment/refund was not immediate")
	}
	g.EndTurn()
	line := woodTurnReport(g, p)
	if line["start"] != float64(120) || line["spent"] != float64(120) || line["gathered"] != float64(60) || line["final"] != float64(60) {
		t.Fatalf("ordering transactions missing from report: %+v", line)
	}
	if turn, _ := g.Risq.Standings(); turn != 1 {
		t.Fatalf("standings advanced to unfinished turn %d", turn)
	}
	g.Action(p, "set-auto-renew", gin.H{"building_id": 3, "count": 2})
	if report := woodTurnReport(g, p); report["spent"] != float64(120) || report["gathered"] != float64(60) {
		t.Fatalf("new ordering phase changed completed report: %+v", report)
	}
	g.EndTurn()
	line = woodTurnReport(g, p)
	if line["start"] != float64(60) || line["spent"] != float64(60) || line["gathered"] != float64(0) || line["final"] != float64(0) {
		t.Fatalf("next turn reused old counters: %+v", line)
	}
	g.EndTurn()
	line = woodTurnReport(g, p)
	if line["spent"] != float64(0) || line["gathered"] != float64(0) {
		t.Fatalf("empty turn retained resource flows: %+v", line)
	}
}
