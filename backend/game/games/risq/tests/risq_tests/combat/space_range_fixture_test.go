package combat

import (
	"fmt"
	"strings"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

func spaceRangeGame(t *testing.T, attackRange defs.RisqRange, spaces ...string) *harness.Game {
	t.Helper()
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"space-range": fmt.Sprintf(meetingBoard, strings.Join(spaces, ","))})
	for _, id := range []uint32{blunt, heavy} {
		config := defs.UnitConfigs[id]
		config.Attack_range = attackRange
		defs.UnitConfigs[id] = config
	}
	return harness.NewGame(t, "custom:space-range", 1, 2)
}

func spaceRangeAttackOrder(attacker, target harness.Unit, kind defs.OrderType) defs.OrderFromFrontend {
	targetID := int64(target.InternalID)
	if kind == defs.OrderType_UnitAttackZone {
		targetID = harness.ZoneKey(target.Space.X, target.Space.Y, target.Zone.X, target.Zone.Y)
	} else if kind == defs.OrderType_UnitAttackSpace {
		targetID = harness.SpaceKey(target.Space.X, target.Space.Y)
	}
	return harness.Order(kind, []uint64{attacker.InternalID}, targetID, false)
}

func TestPositiveRangeShootsAcrossBorder(t *testing.T) {
	for name, attackRange := range map[string]defs.RisqRange{"adjacent": defs.RisqRange_ADJACENT, "secondary": defs.RisqRange_SECONDARY} {
		t.Run(name, func(t *testing.T) {
			log := captureTicks(t)
			g := spaceRangeGame(t, attackRange, acrossSpaces(placed{heavy, 0, 1, 0}, placed{heavy, 1, -1, 0})...)
			a, b := soleUnit(g, 0), soleUnit(g, 1)
			g.Submit(g.Human(0), attackOrder(a, b))
			g.Submit(g.Human(1), attackOrder(b, a))
			ticks := parseTicks(t, log)
			if len(ticks.moves) != 0 || len(ticks.meetings) != 0 || len(ticks.hitsBy(int(a.InternalID))) == 0 || len(ticks.hitsBy(int(b.InternalID))) == 0 {
				t.Fatalf("positive range did not shoot normally: %+v", ticks)
			}
		})
	}
}

func TestSpaceRangeShootsWithinSpace(t *testing.T) {
	for name, kind := range map[string]defs.OrderType{"unit": defs.OrderType_UnitAttackUnit, "zone": defs.OrderType_UnitAttackZone, "space": defs.OrderType_UnitAttackSpace} {
		t.Run(name, func(t *testing.T) {
			log := captureTicks(t)
			g := spaceRangeGame(t, defs.RisqRange_SPACE, spaceOf(0, 0, flat, placed{heavy, 0, 0, 0}, placed{heavy, 1, 1, 0}))
			a, b := soleUnit(g, 0), soleUnit(g, 1)
			g.Submit(g.Human(0), spaceRangeAttackOrder(a, b, kind))
			g.Submit(g.Human(1), spaceRangeAttackOrder(b, a, kind))
			ticks := parseTicks(t, log)
			if len(ticks.moves) != 0 || len(ticks.meetings) != 0 {
				t.Fatalf("same-space shooting moved or met: %+v", ticks)
			}
			for _, unit := range []harness.Unit{a, b} {
				hits := ticks.hitsBy(int(unit.InternalID))
				if len(hits) == 0 || hits[0].tick != 1 || hits[0].stamina != 3 {
					t.Errorf("unit %d attacks %+v, want full-strength shooting on tick 1", unit.InternalID, hits)
				}
			}
		})
	}
}

func TestSpaceRangeCrossingAttacks(t *testing.T) {
	for name, kind := range map[string]defs.OrderType{"unit": defs.OrderType_UnitAttackUnit, "zone": defs.OrderType_UnitAttackZone, "space": defs.OrderType_UnitAttackSpace} {
		t.Run(name, func(t *testing.T) {
			log := captureTicks(t)
			g := spaceRangeGame(t, defs.RisqRange_SPACE, acrossSpaces(placed{heavy, 0, 1, 0}, placed{heavy, 1, -1, 0})...)
			a, b := soleUnit(g, 0), soleUnit(g, 1)
			g.Submit(g.Human(0), spaceRangeAttackOrder(a, b, kind))
			g.Submit(g.Human(1), spaceRangeAttackOrder(b, a, kind))
			ticks := parseTicks(t, log)
			if len(ticks.moves) != 0 {
				t.Fatalf("units crossed instead of meeting: %+v", ticks.moves)
			}
			for slot, before := range []harness.Unit{a, b} {
				after := soleUnit(g, slot)
				if after.Space != before.Space || after.Zone != before.Zone || after.CombatStats.Health >= before.CombatStats.Health {
					t.Errorf("slot %d did not hold position and exchange damage: %+v", slot, after)
				}
				first, found := ticks.meetingOf(int(before.InternalID))
				if !found || first.tick != 1 || first.sunk != 3 || first.attack != 0 {
					t.Errorf("slot %d first meeting %+v found=%v, want a 3-stamina half step", slot, first, found)
				}
			}
		})
	}
}

func TestSpaceRangeApproachesStationaryEnemy(t *testing.T) {
	for name, kind := range map[string]defs.OrderType{"unit": defs.OrderType_UnitAttackUnit, "zone": defs.OrderType_UnitAttackZone, "space": defs.OrderType_UnitAttackSpace} {
		t.Run(name, func(t *testing.T) {
			log := captureTicks(t)
			g := spaceRangeGame(t, defs.RisqRange_SPACE, acrossSpaces(placed{heavy, 0, 1, 0}, placed{heavy, 1, -1, 0})...)
			a, b := soleUnit(g, 0), soleUnit(g, 1)
			makePassive(g, 1, b)
			g.Submit(g.Human(0), spaceRangeAttackOrder(a, b, kind))
			g.Submit(g.Human(1))
			ticks := parseTicks(t, log)
			if len(ticks.meetings) != 0 || len(ticks.movesOf(int(a.InternalID))) != 1 || len(ticks.hitsBy(int(a.InternalID))) == 0 {
				t.Fatalf("stationary enemy did not get approached and shot: %+v", ticks)
			}
		})
	}
}

func TestSpaceRangeDoesNotMeetNonOpposingMove(t *testing.T) {
	log := captureTicks(t)
	g := spaceRangeGame(t, defs.RisqRange_SPACE, acrossSpaces(placed{heavy, 0, 1, 0}, placed{heavy, 1, -1, 0})...)
	a, b := soleUnit(g, 0), soleUnit(g, 1)
	makePassive(g, 1, b)
	g.Submit(g.Human(0), attackOrder(a, b))
	g.Submit(g.Human(1), harness.OrderMoveZone([]uint64{b.InternalID}, 1, 0, 0, 0))
	ticks := parseTicks(t, log)
	if len(ticks.meetings) != 0 || len(ticks.movesOf(int(a.InternalID))) != 1 || len(ticks.hitsBy(int(a.InternalID))) == 0 {
		t.Fatalf("non-opposing movement incorrectly met or stopped shooting: %+v", ticks)
	}
}
