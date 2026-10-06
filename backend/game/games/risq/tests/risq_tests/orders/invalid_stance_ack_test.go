package orders

import (
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/gin-gonic/gin"
)

func TestInvalidStanceNotificationsMatchAuthoritativeUnits(t *testing.T) {
	for _, stance := range []uint8{0, uint8(defs.UnitStance_END), 255} {
		t.Run(fmt.Sprint(stance), func(t *testing.T) {
			g := notificationGame(t, true)
			p, spy := g.Human(0), g.Human(1)
			ids := unitIDs(g, p, 11)
			g.Action(p, "set-unit-behavior", gin.H{"internal_ids": ids[:1], "stance": 2})
			g.Action(p, "set-unit-behavior", gin.H{"internal_ids": ids[1:], "stance": 4})
			stances := map[uint64]any{ids[0]: entity(g, p, "units", ids[0])["stance"], ids[1]: entity(g, p, "units", ids[1])["stance"]}
			clients := map[int]gin.H{p: g.Risq.ToFrontend(uint64(p+1), false), spy: g.Risq.ToFrontend(uint64(spy+1), false)}
			rawAction(g, p, "set-unit-behavior", gin.H{"internal_ids": ids, "stance": stance, "attack_back": false})
			for _, recipient := range []int{p, spy} {
				update := nextUpdate(t, g.Base.Players[uint64(recipient+1)].Updates)
				affected := update.Content["internal_ids"].([]uint64)
				if update.Kind != "unit-behavior-set" || (recipient == p && len(affected) != 2) || (recipient == spy && len(affected) != 1) {
					t.Fatalf("invalid stance notification recipients: %+v", update)
				}
				for _, id := range update.Content["internal_ids"].([]uint64) {
					unit := entity(g, p, "units", id)
					if snapshotJSON(t, unit["stance"]) != snapshotJSON(t, stances[id]) {
						t.Fatal("invalid stance changed authoritative state")
					}
					if value, exists := update.Content["stance"]; exists && snapshotJSON(t, value) != snapshotJSON(t, unit["stance"]) {
						t.Fatalf("stance acknowledgement %v differs from unit %v", value, unit["stance"])
					}
				}
				clients[recipient] = applyClientUpdate(t, clients[recipient], update)
				if snapshotJSON(t, clients[recipient]) != snapshotJSON(t, g.Risq.ToFrontend(uint64(recipient+1), false)) {
					t.Fatal("invalid stance notification diverged from authoritative client state")
				}
			}
		})
	}
}
