package world

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

var origin = harness.Coord{}

func TestTickHistoryInsufficientStamina(t *testing.T) {
	t.Run("cancelled-and-completed-orders", func(t *testing.T) {
		g, p := moverGame(t, spaceWith(2, 2, grass, ""), spaceWith(1, 0, grass, ""))
		u := firstUnit(g, p)
		g.Submit(p, harness.OrderMove([]uint64{u.InternalID}, 2, 2), harness.OrderMove([]uint64{u.InternalID}, 1, 0))
		g.EndTurn()
		after := g.State(p).Unit(u.InternalID)
		if after.Space != (harness.Coord{X: 1}) || len(after.ActiveOrders) != 0 {
			t.Fatal("fixture did not cancel and complete its queue")
		}
		rejected := harness.RequireTickBlocked(t, after.TickActions, "no_reachable_target")
		if rejected.Order.TargetID != harness.SpaceKey(2, 2) {
			t.Errorf("cancelled order target lost: %+v", rejected)
		}
		harness.AssertTickSpend(t, after.TickActions, 7)
		completed := false
		for _, a := range after.TickActions {
			if a.Intent.Kind == "move" && a.Order.TargetID == harness.SpaceKey(1, 0) && a.Execute.Outcome == "executed" {
				completed = true
			}
		}
		if !completed {
			t.Fatal("completed movement disappeared with its order")
		}
	})
	g, p := moverGame(t, spaceWith(1, 0, hills, ""))
	u := firstUnit(g, p)
	g.Submit(p, harness.OrderMove([]uint64{u.InternalID}, 1, 0))
	g.EndTurn()
	after := g.State(p).Unit(u.InternalID)
	if after.Space != origin || len(after.ActiveOrders) != 1 {
		t.Fatal("fixture did not retain an unaffordable crossing")
	}
	blocked := harness.RequireTickBlocked(t, after.TickActions, "insufficient_stamina")
	if blocked.Intent.Kind != "move" || blocked.Intent.MinCost <= blocked.Intent.AvailableStamina || blocked.Tick < 1 {
		t.Errorf("discarded movement proposal missing: %+v", blocked)
	}
	harness.AssertTickSpend(t, after.TickActions, 1)
	replay := harness.RequireReplay(t, g.State(p))
	settled := replay.UnitAt(u.InternalID, replay.TickCount)
	if settled == nil || settled.CurrentStamina != u.CurrentStamina-1 || after.CurrentStamina != 12 {
		t.Fatal("movement history lost capped regeneration boundary")
	}
	if frame := replay.Frame(t, blocked.Tick); !frame.Terminal {
		t.Errorf("last rejected attempt is not terminal: %+v", frame)
	}
}

// P0's lone villager at (0,0) and P1's lone bystander at (3,-3), with the given space entries between
func moverGame(t *testing.T, spaces ...string) (*harness.Game, int) {
	t.Helper()
	all := spaceWith(0, 0, grass, unit(villager, 0)) + "," + spaceWith(3, -3, grass, unit(villager, 1))
	for _, s := range spaces {
		all += "," + s
	}
	g := startGame(t, mapDoc("", all))
	return g, g.Human(0)
}

func TestBorderCrossingTurnsFollowTerrainCost(t *testing.T) {
	cases := map[string]struct {
		terrain, turns int
		why            string
	}{
		"flat": {grass, 1, "a villager has 8 stamina and the crossing costs 1 + 6"},
		"hill": {hills, 2, "the 9-stamina crossing is only affordable with carried-over stamina"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			g, mover := moverGame(t, spaceWith(1, 0, c.terrain, ""))
			if turns := moveUntilArrived(g, mover, 1, 0, 4); turns != c.turns {
				t.Errorf("arrived after %d turns, want %d: %s", turns, c.turns, c.why)
			}
		})
	}
}

func TestMoveRoutesAroundImpassableSpaces(t *testing.T) {
	g, mover := moverGame(t, spaceWith(1, 0, water, ""), spaceWith(1, -1, grass, ""), spaceWith(2, -1, grass, ""), spaceWith(2, 0, grass, ""))
	g.Submit(mover, harness.OrderMove([]uint64{firstUnit(g, mover).InternalID}, 2, 0))
	planned := 0
	for turn := 0; turn < 6 && where(g, mover) != (harness.Coord{X: 2, Y: 0}); turn++ {
		g.EndTurn()
		for _, step := range firstUnit(g, mover).MovePath {
			planned++
			if step.Space == (harness.Coord{X: 1, Y: 0}) {
				t.Fatalf("the planned route %v passes through the water space", firstUnit(g, mover).MovePath)
			}
		}
	}
	if at := where(g, mover); at != (harness.Coord{X: 2, Y: 0}) {
		t.Errorf("unit at %v, want it to have walked around the water to (2,0)", at)
	}
	if planned == 0 {
		t.Error("no route was ever reported, so the detour was not observed")
	}
}

func TestUnreachableMoveIsCancelledWithoutSpendingStamina(t *testing.T) {
	g, mover := moverGame(t, spaceWith(2, 2, grass, ""), spaceWith(1, 0, grass, ""))
	id := firstUnit(g, mover).InternalID
	g.Submit(mover, harness.OrderMove([]uint64{id}, 2, 2), harness.OrderMove([]uint64{id}, 1, 0))
	g.EndTurn()
	if orders := g.Self(mover).ActiveOrders; len(orders) != 0 {
		t.Errorf("%d orders still queued, want the unreachable move cancelled and valid move finished", len(orders))
	}
	if got := firstUnit(g, mover).CurrentStamina; got != 9 {
		t.Errorf("stamina %d, want 9: 8 starting minus 7 for movement plus 8 refreshed", got)
	}
	if at := where(g, mover); at != (harness.Coord{X: 1}) {
		t.Errorf("unit at %v, want the queued valid move to finish at (1,0)", at)
	}
}

func TestMovesTheEngineCannotMakeAreDropped(t *testing.T) {
	cases := map[string]struct {
		target  harness.Coord
		terrain int
	}{
		"impassable":   {harness.Coord{X: 1, Y: 0}, water},
		"disconnected": {harness.Coord{X: 2, Y: 2}, grass},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			g, mover := moverGame(t, spaceWith(c.target.X, c.target.Y, c.terrain, ""))
			if moveUntilArrived(g, mover, c.target.X, c.target.Y, 3) != -1 || where(g, mover) != origin {
				t.Errorf("unit at %v, want it to stay at the origin", where(g, mover))
			}
			if stamina := firstUnit(g, mover).CurrentStamina; stamina != 12 {
				t.Errorf("stamina %d after 3 idle turns, want the cap of 12: a dropped order spends nothing", stamina)
			}
		})
	}
}
