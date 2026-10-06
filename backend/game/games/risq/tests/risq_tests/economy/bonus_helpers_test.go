package economy

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func bonusDamage(t *testing.T, building, enabled bool) float64 {
	t.Helper()
	source := `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":11,"player":0,"count":1},{"id":12,"player":1,"count":1}]`
	if building {
		source += `,"building":{"id":2,"player":1}`
	}
	g := scenarioGame(t, source+`}]}`)
	original := defs.BonusConfigs
	defer func() { defs.BonusConfigs = original }()
	if !enabled {
		defs.BonusConfigs = nil
	}
	p, enemy := g.Human(0), g.Human(1)
	u := g.Self(p).Units[0]
	if u.CombatStats.AttackBlunt != 6 {
		t.Fatalf("targeted bonus baked into base attack: %d", u.CombatStats.AttackBlunt)
	}
	target := g.Self(enemy).Units[0].InternalID
	health := g.Self(enemy).Units[0].CombatStats.Health
	typ := defs.OrderType_UnitAttackUnit
	if building {
		b := centerHousing(g, enemy)
		target, health, typ = b.InternalID, b.CombatStats.Health, defs.OrderType_UnitAttackBuilding
	}
	g.Submit(p, harness.Order(typ, []uint64{u.InternalID}, int64(target), false))
	g.EndTurn()
	if building {
		return health - centerHousing(g, enemy).CombatStats.Health
	}
	survivor := g.State(enemy).Unit(target)
	if survivor == nil {
		t.Fatal("targeted building bonus killed the unit target")
	}
	return health - survivor.CombatStats.Health
}
