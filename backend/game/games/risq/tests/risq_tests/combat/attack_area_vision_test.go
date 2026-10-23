package combat

import (
	"fmt"
	"strings"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

// A passive grunt at the center of (0,0) and the enemy's phalanx far away at (-2,0), leaving (2,0) empty and out of sight
func unseenSpaceGame(t *testing.T, between int) *harness.Game {
	t.Helper()
	spaces := []string{spaceOf(0, 0, flat, placed{blunt, 0, 0, 0}), spaceOf(1, 0, between), spaceOf(2, 0, flat), spaceOf(-1, 0, flat), spaceOf(-2, 0, flat, placed{heavy, 1, 0, 0})}
	doc := fmt.Sprintf(`{"board_size":3,"players":2,"spaces":[%s]}`, strings.Join(spaces, ","))
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"unseen": doc})
	g := harness.NewGame(t, "custom:unseen", 1, 2)
	makePassive(g, 0, soleUnit(g, 0))
	makePassive(g, 1, soleUnit(g, 1))
	return g
}

// From a center zone the adjacent space is only poorly seen; one step to the facing edge gives good vision mid-turn, so the shooting starts on tick 2
func TestAttackSpaceNeedsGoodVisionBeforeShooting(t *testing.T) {
	log := captureTicks(t)
	g := spaceRangeGame(t, defs.RisqRange_ADJACENT, spaceOf(0, 0, flat, placed{blunt, 0, 0, 0}), spaceOf(1, 0, flat, placed{heavy, 1, 0, 0}))
	a, b := soleUnit(g, 0), soleUnit(g, 1)
	makePassive(g, 1, b)
	g.Submit(g.Human(0), spaceRangeAttackOrder(a, b, defs.OrderType_UnitAttackSpace))
	g.Submit(g.Human(1))
	ticks := parseTicks(t, log)
	hits := ticks.hitsBy(int(a.InternalID))
	if len(ticks.movesOf(int(a.InternalID))) != 1 || len(hits) == 0 || hits[0].tick != 2 {
		t.Fatalf("shot at a unit it could not see well: %+v", ticks)
	}
}

func TestAttackSpaceWalksToUnseenSpaceBeforeJudgingItEmpty(t *testing.T) {
	g := unseenSpaceGame(t, flat)
	owner := g.Human(0)
	g.Submit(owner, harness.Order(defs.OrderType_UnitAttackSpace, []uint64{soleUnit(g, 0).InternalID}, harness.SpaceKey(2, 0), false))
	g.Submit(g.Human(1))
	if unit := soleUnit(g, 0); unit.Space.X < 1 {
		t.Fatalf("order was judged complete without looking: unit still at %+v", unit.Space)
	}
	for turn := 0; turn < 3 && len(g.Self(owner).Units[0].ActiveOrders) > 0; turn++ {
		g.EndTurn()
	}
	if orders := g.Self(owner).Units[0].ActiveOrders; len(orders) != 0 {
		t.Fatalf("order never completed after the space was seen empty: %+v", orders)
	}
	walled := unseenSpaceGame(t, 251)
	walled.Submit(walled.Human(0), harness.Order(defs.OrderType_UnitAttackSpace, []uint64{soleUnit(walled, 0).InternalID}, harness.SpaceKey(2, 0), false))
	walled.Submit(walled.Human(1))
	if orders := soleUnit(walled, 0).ActiveOrders; len(orders) != 0 {
		t.Fatalf("order to a space behind water stayed active: %+v", orders)
	}
}
