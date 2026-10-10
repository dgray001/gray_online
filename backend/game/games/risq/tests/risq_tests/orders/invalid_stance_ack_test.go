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

func TestInvalidDefaultStance(t *testing.T) {
	g := notificationGame(t, false)
	p := g.Human(0)
	g.Action(p, "set-default-unit-stance", gin.H{"stance": uint8(defs.UnitStance_AGGRESSIVE)})
	for _, stance := range []any{nil, 0, uint8(defs.UnitStance_END), 255, -1, "aggressive"} {
		rawAction(g, p, "set-default-unit-stance", gin.H{"stance": stance})
		if update := nextUpdate(t, g.Base.Players[uint64(p+1)].FailedUpdates); update.Kind != "set-default-unit-stance-failed" {
			t.Fatalf("invalid stance accepted: %+v", update)
		}
		for _, player := range g.Risq.ToFrontend(uint64(p+1), false)["players"].([]gin.H) {
			if player["player"].(gin.H)["player_id"] == g.PlayerID(p) && player["default_unit_stance"] != defs.UnitStance_AGGRESSIVE {
				t.Fatal("invalid stance changed the default")
			}
		}
	}
}

func TestDefaultBehaviorPartialUpdatesPreserveExistingUnits(t *testing.T) {
	g := notificationGame(t, false)
	p := g.Human(0)
	id := unitIDs(g, p, 11)[0]
	before := snapshotJSON(t, entity(g, p, "units", id))
	g.Action(p, "set-default-unit-behavior", gin.H{"stance": 2, "attack_back": false, "interrupt_current": true, "target_priority": []uint8{1}})
	rawAction(g, p, "set-default-unit-behavior", gin.H{"target_priority": []uint8{}})
	update := latestUpdate(t, g.Base.Players[uint64(p+1)].Updates)
	if update.Kind != "default-unit-behavior-set" || update.Content["stance"] != defs.UnitStance(2) || update.Content["attack_back"] != false || update.Content["interrupt_current"] != true || len(update.Content["target_priority"].([]int)) != 0 {
		t.Fatalf("partial update lost defaults: %+v", update)
	}
	if snapshotJSON(t, entity(g, p, "units", id)) != before {
		t.Fatal("default behavior changed an existing unit")
	}
}
