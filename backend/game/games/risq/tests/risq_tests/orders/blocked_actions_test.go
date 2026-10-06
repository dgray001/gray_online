package orders

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

func blockedActions(g *harness.Game, p int) map[string]gin.H {
	order := harness.OrderMove(unitIDs(g, p, 1)[:1], 2, 0)
	order.Player_id = g.PlayerID(p)
	return map[string]gin.H{
		"submit-orders":         {"orders": []defs.OrderFromFrontend{order}},
		"set-unit-behavior":     {"internal_ids": unitIDs(g, p, 11), "stance": 2, "attack_back": false, "interrupt_current": true},
		"set-building-behavior": {"internal_ids": []uint64{buildingID(g, p, 23)}, "auto_attack": false, "interrupt_current": true},
		"set-gather-point":      {"building_id": buildingID(g, p, 1), "location_kind": 1, "location_id": harness.SpaceKey(3, 0)},
	}
}
