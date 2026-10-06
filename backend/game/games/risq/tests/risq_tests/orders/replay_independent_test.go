package orders

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

func TestReplayRestoresIndependentDroppedChanges(t *testing.T) {
	for _, last := range []string{"unit", "building", "gather-point", "other-submit"} {
		t.Run(last, func(t *testing.T) {
			g := orderGame(t, richBank, "")
			p := g.Human(0)
			player := g.Base.Players[uint64(p+1)]
			latestUpdate(t, player.Updates)
			client := g.Risq.ToFrontend(uint64(p+1), false)
			for range cap(player.Updates) {
				rawAction(g, p, "set-unit-behavior", gin.H{"internal_ids": unitIDs(g, p, 11), "stance": 1})
			}
			rawAction(g, p, "set-unit-behavior", gin.H{"internal_ids": unitIDs(g, p, 11), "attack_back": false})
			rawAction(g, p, "set-building-behavior", gin.H{"internal_ids": []uint64{buildingID(g, p, 23)}, "auto_attack": false})
			rawAction(g, p, "set-gather-point", gin.H{"building_id": buildingID(g, p, 1), "location_kind": 1, "location_id": harness.SpaceKey(3, 0)})
			switch last {
			case "unit":
				rawAction(g, p, "set-unit-behavior", gin.H{"internal_ids": unitIDs(g, p, 11), "stance": 4})
			case "building":
				rawAction(g, p, "set-building-behavior", gin.H{"internal_ids": []uint64{buildingID(g, p, 23)}, "interrupt_current": true})
			case "gather-point":
				rawAction(g, p, "set-gather-point", gin.H{"building_id": buildingID(g, p, 1), "clear": true})
			case "other-submit":
				rawAction(g, g.Human(1), "submit-orders", nil)
			}
			for len(player.Updates) > 0 {
				client = applyClientUpdate(t, client, nextUpdate(t, player.Updates))
			}
			expected := snapshotJSON(t, g.Risq.ToFrontend(uint64(p+1), false))
			if snapshotJSON(t, client) == expected {
				t.Fatal("overflow fixture did not lose independent changes")
			}
			g.Base.ResendLastUpdate(uint64(p + 1))
			client = applyClientUpdate(t, client, nextUpdate(t, player.Updates))
			if snapshotJSON(t, client) != expected {
				t.Fatal("replay did not restore the complete client snapshot")
			}
		})
	}
}
