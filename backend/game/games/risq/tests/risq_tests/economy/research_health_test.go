package economy

import (
	"math"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestResearchPreservesDamagedUnitHealthRatio(t *testing.T) {
	g := economyGame(t, `{"food":100,"wood":100}`, `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":1,"player":0},"units":[{"id":1,"player":0,"count":1},{"id":1,"player":1,"count":1}]}]}`, "")
	p, enemy := g.Human(0), g.Human(1)
	id := g.Self(p).Units[0].InternalID
	g.Submit(enemy, harness.OrderAttackUnit([]uint64{g.Self(enemy).Units[0].InternalID}, id))
	g.EndTurn()
	u := g.State(p).Unit(id)
	if u == nil || u.CombatStats.Health <= 0 || u.CombatStats.Health >= float64(u.CombatStats.MaxHealth) {
		t.Fatal("combat did not leave a damaged surviving villager")
	}
	for _, human := range []int{enemy, p} {
		orders := []defs.OrderFromFrontend{}
		for _, active := range g.Self(human).ActiveOrders {
			orders = append(orders, harness.Order(defs.OrderType_CancelOrder, nil, int64(active.InternalID), false))
		}
		if human == p {
			orders = append(orders, harness.Order(defs.OrderType_BuildingResearch, []uint64{g.Self(p).Buildings[0].InternalID}, 1, false))
		}
		g.Submit(human, orders...)
	}
	g.EndTurn()
	after := g.State(p).Unit(id)
	want := u.CombatStats.Health * 14 / float64(u.CombatStats.MaxHealth)
	if after == nil || !g.Self(p).ResearchedTechs[1] || after.CombatStats.MaxHealth != 14 || math.Abs(after.CombatStats.Health-want) > 0.0001 {
		t.Errorf("research did not preserve health ratio: unit %+v, want health %v", after, want)
	}
}
