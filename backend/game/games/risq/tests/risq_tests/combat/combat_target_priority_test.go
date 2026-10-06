package combat

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
	"github.com/gin-gonic/gin"
	"testing"
)

func TestTargetPriority(t *testing.T) {
	// p0 has unit 13. p1 has 10 unit 1 (Villager, Economic) and unit 11 (Blunt Infantry, Infantry).
	doc := `{"board_size":1,"players":2,"spaces":[{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":13,"player":0,"count":1},{"id":1,"player":1,"count":10},{"id":11,"player":1,"count":1}]}]}]}`
	fakeboard.UseConfig(t, shippedConfig, nil, map[string]string{"combat_priority": doc})
	g := harness.NewGame(t, "custom:combat_priority", 1, 2)
	p0, p1 := g.Human(0), g.Human(1)
	u0 := g.Self(p0).Units[0]

	// Set target priority to prefer ECONOMIC over INFANTRY
	g.Action(p0, "set-unit-behavior", gin.H{
		"internal_ids":    []uint64{u0.InternalID},
		"target_priority": []uint8{uint8(defs.TargetCategory_ECONOMIC), uint8(defs.TargetCategory_MILITARY)},
	})

	// Order attack on space
	g.Submit(p0, harness.Order(defs.OrderType_UnitAttackSpace, []uint64{u0.InternalID}, harness.SpaceKey(0, 0), false))
	g.Submit(p1)

	var infantry *harness.Unit
	survivors := g.Self(p1).Units
	for i := range survivors {
		if survivors[i].UnitID == 11 {
			infantry = &survivors[i]
		}
	}
	if infantry == nil {
		t.Errorf("infantry should have survived, but it was killed")
	} else if infantry.CombatStats.Health < 11 {
		t.Errorf("infantry incorrectly took damage, health %v", infantry.CombatStats.Health)
	}
}
