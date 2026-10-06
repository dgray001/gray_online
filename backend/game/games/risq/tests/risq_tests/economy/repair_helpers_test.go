package economy

import (
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func damagedHousingGame(t *testing.T) (*harness.Game, int) {
	t.Helper()
	return damagedRepairGame(t, `{"wood":120}`, 1)
}

func damagedRepairGame(t *testing.T, bank string, workers int) (*harness.Game, int) {
	t.Helper()
	space := fmt.Sprintf(`{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":2,"player":0},"units":[{"id":1,"player":0,"count":%d},{"id":1,"player":1,"count":8}]}]}`, workers)
	g := economyGame(t, bank, space, "")
	p, enemy := g.Human(0), g.Human(1)
	id := centerHousing(g, p).InternalID
	attackers := []uint64{}
	for _, u := range g.Self(enemy).Units {
		attackers = append(attackers, u.InternalID)
	}
	g.Submit(enemy, harness.Order(defs.OrderType_UnitAttackBuilding, attackers, int64(id), false))
	for turn := 0; centerHousing(g, p).CombatStats.Health >= 50 && turn < 40; turn++ {
		g.EndTurn()
	}
	health := centerHousing(g, p).CombatStats.Health
	orders := g.Self(enemy).ActiveOrders
	if health >= 50 || health <= 0 || len(orders) != 1 {
		t.Fatalf("damage setup: health %v, enemy orders %v", health, orders)
	}
	g.Submit(enemy, harness.Order(defs.OrderType_CancelOrder, nil, int64(orders[0].InternalID), false))
	g.EndTurn()
	if got := centerHousing(g, p).CombatStats.Health; got != health {
		t.Fatalf("cancelled attacker changed health from %v to %v", health, got)
	}
	return g, p
}
