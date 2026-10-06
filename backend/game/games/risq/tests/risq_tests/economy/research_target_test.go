package economy

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestResearchLeavesNonmatchingAndEnemyUnitsUnchanged(t *testing.T) {
	g := economyGame(t, `{"food":100,"wood":100}`, `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":1,"player":0},"units":[{"id":1,"player":0,"count":1},{"id":11,"player":0,"count":1}]}]},`+remoteVillager, "")
	p, enemy := g.Human(0), g.Human(1)
	before := g.Self(p).Units
	enemyBefore := g.Self(enemy).Units[0]
	g.Submit(p, harness.Order(defs.OrderType_BuildingResearch, []uint64{g.Self(p).Buildings[0].InternalID}, 1, false))
	g.EndTurn()
	g.EndTurn()
	if !g.Self(p).ResearchedTechs[1] {
		t.Fatal("farming did not finish")
	}
	for _, u := range before {
		if u.UnitID == 11 {
			after := g.State(p).Unit(u.InternalID)
			if after == nil || after.CombatStats != u.CombatStats || after.TurnStamina != u.TurnStamina {
				t.Error("farming affected nonmatching infantry")
			}
		}
	}
	if after := g.Self(enemy).Units[0]; after.CombatStats != enemyBefore.CombatStats || after.TurnStamina != enemyBefore.TurnStamina || g.Self(enemy).ResearchedTechs[1] {
		t.Error("farming affected the enemy")
	}
}
