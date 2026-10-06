package shape

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestCombatReportWireShapeForBothPlayers(t *testing.T) {
	g := configuredGame(t, 3, func() {
		unit := defs.UnitConfigs[11]
		unit.Attack_range, unit.Attack_blunt = defs.RisqRange_ADJACENT, 200
		defs.UnitConfigs[11] = unit
	})
	attacker, victim := unitID(g, 1, 0, 11), unitID(g, 0, -1, 15)
	g.Submit(g.Human(1), harness.OrderAttackUnit([]uint64{attacker}, victim))
	g.EndTurn()
	for slot := range 2 {
		p := player(t, snapshot(g, g.Human(slot), false), slot)
		events := array(t, object(t, p["turn_report"])["combat"])
		equal(t, len(events), 1)
		event := checkShape(t, events[0], "tick:n kind:n self_player:n other_player:n target_id:n space:o zone:o damage:n")
		checkShape(t, event["space"], coordinateShape)
		checkShape(t, event["zone"], coordinateShape)
		equal(t, event["self_player"], float64(slot))
		equal(t, event["other_player"], float64(1-slot))
		equal(t, event["kind"], float64(4-slot))
		equal(t, event["target_id"], float64(15))
		if event["damage"].(float64) <= 0 {
			t.Error("combat event has no damage")
		}
	}
	if g.State(g.Human(0)).Unit(victim) != nil {
		t.Error("dead unit remains in owner snapshot")
	}
}
