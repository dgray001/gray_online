package world

import (
	"fmt"
	"strings"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

var origin = harness.Coord{}

func TestReplayEnemyLeavesGoodVision(t *testing.T) {
	observer := defs.UnitConfigs[worker]
	observer.Vision.Adjacent, observer.Vision.Secondary = defs.VisibilityGood, defs.VisibilityPoor
	defs.UnitConfigs[15] = observer
	fast := defs.UnitConfigs[worker]
	fast.Turn_stamina = 30
	defs.UnitConfigs[worker] = fast
	spaces := spaceWith(0, 0, flatTerrain, unit(15, 0)) + "," + spaceWith(1, 0, flatTerrain, unit(worker, 1)) + "," + spaceWith(2, 0, flatTerrain, "")
	g := startGame(t, mapDoc("", spaces))
	owner, viewer := g.Human(1), g.Human(0)
	id := firstUnit(g, owner).InternalID
	g.Submit(owner, harness.OrderMove([]uint64{id}, 2, 0))
	g.EndTurn()
	replay := harness.RequireReplay(t, g.State(viewer))
	zone := firstUnit(g, owner).Zone
	if replay.UnitAt(id, 0) == nil || replay.UnitAt(id, replay.TickCount) != nil {
		t.Fatal("enemy should be visible initially and hidden after leaving good vision")
	}
	counted := false
	for _, frame := range replay.Frames {
		for _, space := range frame.Spaces {
			if space.CoordinateKey == harness.SpaceKey(2, 0) && space.Visibility == poor && space.UnitCounts != nil {
				row := zone.Y + 1
				column := zone.X + min(1, zone.Y+1)
				counted = counted || space.UnitCounts[row][column] == 1
			}
		}
	}
	if !counted {
		t.Fatal("replay never records the enemy as an anonymous count in its destination zone")
	}
	g.Submit(owner, harness.OrderMove([]uint64{id}, 1, 0))
	g.EndTurn()
	replay = harness.RequireReplay(t, g.State(viewer))
	if replay.UnitAt(id, 0) != nil || replay.UnitAt(id, replay.TickCount) == nil {
		t.Fatal("enemy should regain individual visibility when returning from poor vision")
	}
}

func TestTickHistoryInsufficientStamina(t *testing.T) {
	t.Run("cancelled-and-completed-orders", func(t *testing.T) {
		g, p := moverGame(t, spaceWith(2, 2, flatTerrain, ""), spaceWith(1, 0, flatTerrain, ""))
		u := firstUnit(g, p)
		g.Submit(p, harness.OrderMove([]uint64{u.InternalID}, 2, 2), harness.OrderMove([]uint64{u.InternalID}, 1, 0))
		g.EndTurn()
		after := g.State(p).Unit(u.InternalID)
		if after.Space != (harness.Coord{X: 1}) || len(after.ActiveOrders) != 0 {
			t.Fatal("fixture did not cancel and complete its queue")
		}
		closed := false
		for _, a := range after.TickActions {
			closed = closed || (a.Order.TargetID == harness.SpaceKey(2, 2) && a.Execute.Reason == "order_completed")
		}
		if !closed {
			t.Error("move that cannot get closer should be recorded as completed")
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
	g, p := moverGame(t, spaceWith(1, 0, hillyTerrain, ""))
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

// P0's lone worker at (0,0) and P1's lone bystander at (3,-3), with the given space entries between
func moverGame(t *testing.T, spaces ...string) (*harness.Game, int) {
	t.Helper()
	slow := defs.UnitConfigs[worker]
	slow.Turn_stamina = 8
	defs.UnitConfigs[worker] = slow
	all := spaceWith(0, 0, flatTerrain, unit(worker, 0)) + "," + spaceWith(3, -3, flatTerrain, unit(worker, 1))
	for _, s := range spaces {
		all += "," + s
	}
	g := startGame(t, mapDoc("", all))
	return g, g.Human(0)
}

func wallSpace(x, y int, units string) string {
	zones := fmt.Sprintf(`{"x":-1,"y":0,"building":{"id":%d,"player":1}}`, wall)
	if units != "" {
		zones += `,{"x":0,"y":0,"units":[` + units + `]}`
	}
	return fmt.Sprintf(`{"x":%d,"y":%d,"terrain":%d,"zones":[%s]}`, x, y, flatTerrain, zones)
}

func TestWallBlocksOnlyEnemiesEnteringFromAnotherSpace(t *testing.T) {
	g := startGame(t, mapDoc("", spaceWith(0, 0, flatTerrain, unit(worker, 0)+","+unit(worker, 1))+","+wallSpace(1, 0, "")))
	enemy, owner := g.Human(0), g.Human(1)
	if turns := moveUntilArrived(g, enemy, 1, 0, 4); turns != -1 {
		t.Errorf("enemy arrived after %d turns, want the wall to block the only crossing", turns)
	}
	if turns := moveUntilArrived(g, owner, 1, 0, 4); turns != 1 {
		t.Errorf("owner arrived after %d turns, want 1", turns)
	}
}

func TestEnemyEntersWallZoneFromWithinItsSpace(t *testing.T) {
	g := startGame(t, mapDoc("", spaceWith(3, -3, flatTerrain, unit(worker, 1))+","+wallSpace(1, 0, unit(worker, 0))))
	enemy := g.Human(0)
	g.Submit(enemy, harness.OrderMoveZone([]uint64{firstUnit(g, enemy).InternalID}, 1, 0, -1, 0))
	g.EndTurn()
	if zone := firstUnit(g, enemy).Zone; zone != (harness.Coord{X: -1, Y: 0}) {
		t.Errorf("the worker is in zone %v, want it to have entered the wall zone (-1,0) from inside the space", zone)
	}
}

const wallHealthFull = 150.0

var borderZone = harness.Coord{X: 1, Y: 0}

func standoffIn(t *testing.T, enemySpace, wallSpaceDoc string, extra ...string) (*harness.Game, int, int) {
	t.Helper()
	sight := defs.UnitConfigs[grunt]
	sight.Vision.Adjacent = defs.VisibilityGood
	defs.UnitConfigs[grunt] = sight
	spaces := append([]string{enemySpace, wallSpaceDoc, spaceWith(3, -3, flatTerrain, unit(worker, 1))}, extra...)
	g := startGame(t, mapDoc("", strings.Join(spaces, ",")))
	return g, g.Human(0), g.Human(1)
}

func standoff(t *testing.T, wallUnits string, extra ...string) (*harness.Game, int, int) {
	t.Helper()
	return standoffIn(t, spaceWith(0, 0, flatTerrain, unit(grunt, 0)), wallSpace(1, 0, wallUnits), extra...)
}

func borderEnemySpace() string {
	return `{"x":0,"y":0,"terrain":1,"zones":[{"x":1,"y":0,"units":[` + unit(grunt, 0) + `]}]}`
}

func wallZoneSpace(zoneUnits string) string {
	return fmt.Sprintf(`{"x":1,"y":0,"terrain":%d,"zones":[{"x":-1,"y":0,"building":{"id":%d,"player":1},"units":[%s]}]}`, flatTerrain, wall, zoneUnits)
}

func findWall(g *harness.Game, owner int) harness.Building {
	g.T.Helper()
	for _, b := range g.Self(owner).Buildings {
		if b.BuildingID == wall {
			return b
		}
	}
	g.T.Fatal("the wall is gone")
	return harness.Building{}
}

func setStance(g *harness.Game, human int, stance defs.UnitStance) {
	g.T.Helper()
	setStanceBack(g, human, stance, false)
}

func setStanceBack(g *harness.Game, human int, stance defs.UnitStance, attackBack bool) {
	g.T.Helper()
	g.Action(human, "set-unit-behavior", gin.H{"internal_ids": []uint64{firstUnit(g, human).InternalID}, "stance": uint8(stance), "attack_back": attackBack})
}

func meetingActions(u harness.Unit) (sunk int, damaging int) {
	for _, a := range u.TickActions {
		if a.Intent.Resolution.SunkCost > 0 {
			sunk++
		}
		for _, e := range a.Execute.Effects {
			if e.Kind == "damage" {
				damaging++
			}
		}
	}
	return sunk, damaging
}

type wallOrder func(wallID, unitID uint64) defs.OrderFromFrontend

var wallAttackOrders = map[string]wallOrder{
	"attack building": func(wallID, unitID uint64) defs.OrderFromFrontend {
		return harness.Order(defs.OrderType_UnitAttackBuilding, []uint64{unitID}, int64(wallID), false)
	},
	"attack zone": func(_, unitID uint64) defs.OrderFromFrontend {
		return harness.Order(defs.OrderType_UnitAttackZone, []uint64{unitID}, harness.ZoneKey(1, 0, -1, 0), false)
	},
	"attack space": func(_, unitID uint64) defs.OrderFromFrontend {
		return harness.Order(defs.OrderType_UnitAttackSpace, []uint64{unitID}, harness.SpaceKey(1, 0), false)
	},
}

func attackWall(g *harness.Game, enemy, owner int, order wallOrder, turns int) harness.Unit {
	g.T.Helper()
	setStance(g, enemy, defs.UnitStance_PASSIVE)
	g.Submit(enemy, order(findWall(g, owner).InternalID, firstUnit(g, enemy).InternalID))
	for range turns {
		g.EndTurn()
	}
	return firstUnit(g, enemy)
}

func requireMetAtBorder(t *testing.T, g *harness.Game, owner int, u harness.Unit) {
	t.Helper()
	if u.Space != origin || u.Zone != borderZone {
		t.Errorf("attacker at space %v zone %v, want it on its own border zone %v", u.Space, u.Zone, borderZone)
	}
	if sunk, damaging := meetingActions(u); sunk != 1 || damaging == 0 {
		t.Errorf("last turn paid %d half moves and dealt damage %d times, want one half move then damage", sunk, damaging)
	}
	if health := findWall(g, owner).CombatStats.Health; health >= wallHealthFull {
		t.Errorf("wall health %v, want it damaged from across the border", health)
	}
}

func requireParkedAtBorder(t *testing.T, g *harness.Game, owner int, u harness.Unit) {
	t.Helper()
	if u.Space != origin || u.Zone != borderZone || len(u.ActiveOrders) != 0 {
		t.Errorf("attacker at space %v zone %v with %d orders, want it walked up to border zone %v with the order ended", u.Space, u.Zone, len(u.ActiveOrders), borderZone)
	}
	if health := findWall(g, owner).CombatStats.Health; health != wallHealthFull {
		t.Errorf("wall health %v, want it untouched without an attack order on it", health)
	}
}

func attackUnit(target uint64) wallOrder {
	return func(_, unitID uint64) defs.OrderFromFrontend {
		return harness.Order(defs.OrderType_UnitAttackUnit, []uint64{unitID}, int64(target), false)
	}
}

func moveToWallZone(uint64) wallOrder {
	return func(_, unitID uint64) defs.OrderFromFrontend {
		return harness.OrderMoveZone([]uint64{unitID}, 1, 0, -1, 0)
	}
}

func requireStandGround(t *testing.T) {
	for name, attacks := range map[string]bool{"center zone": false, "border zone": true} {
		t.Run(name, func(t *testing.T) {
			enemySpace := spaceWith(0, 0, flatTerrain, unit(grunt, 0))
			if attacks {
				enemySpace = borderEnemySpace()
			}
			g, enemy, owner := standoffIn(t, enemySpace, wallSpace(1, 0, ""))
			setStance(g, enemy, defs.UnitStance_STAND_GROUND)
			g.EndTurn()
			g.EndTurn()
			if damaged := findWall(g, owner).CombatStats.Health < wallHealthFull; damaged != attacks {
				t.Errorf("wall damaged is %v, want %v for stand ground at the %s", damaged, attacks, name)
			}
			if zone := firstUnit(g, enemy).Zone; attacks && zone != borderZone {
				t.Errorf("stand ground unit is in zone %v, want it to stay in its zone %v", zone, borderZone)
			}
		})
	}
}

func requireRangedNeedsNoMeeting(t *testing.T) {
	sight := defs.UnitConfigs[longbow]
	sight.Vision.Adjacent, sight.Attack_range = defs.VisibilityGood, defs.RisqRange_ADJACENT
	defs.UnitConfigs[longbow] = sight
	g := startGame(t, mapDoc("", spaceWith(0, 0, flatTerrain, unit(longbow, 0))+","+wallSpace(1, 0, "")+","+spaceWith(3, -3, flatTerrain, unit(worker, 1))))
	u := attackWall(g, g.Human(0), g.Human(1), wallAttackOrders["attack building"], 1)
	if sunk, damaging := meetingActions(u); u.Zone != (harness.Coord{}) || sunk != 0 || damaging == 0 {
		t.Errorf("zone %v, half moves %d, damage %d, want a ranged unit hitting from its own zone with no half move", u.Zone, sunk, damaging)
	}
}

func garrisonedAttackBackGame(t *testing.T) (*harness.Game, int, int) {
	t.Helper()
	armed, limit := defs.BuildingConfigs[wall], 4
	armed.Garrison_capacity, armed.Attack_type, armed.Attack_range, armed.Max_garrison_attack_blunt = 1, defs.AttackType_BLUNT, defs.RisqRange_ADJACENT, &limit
	defs.BuildingConfigs[wall] = armed
	doc := spaceWith(0, 0, flatTerrain, unit(grunt, 0)) + "," + wallZoneSpace(unit(grunt, 1)) + "," + spaceWith(3, -3, flatTerrain, unit(worker, 1))
	g := startGame(t, mapDoc("", doc))
	return g, g.Human(0), g.Human(1)
}

func requireAttackBackAtWall(t *testing.T) {
	for stance, attacksBack := range map[defs.UnitStance]bool{defs.UnitStance_PASSIVE: false, defs.UnitStance_DEFENSIVE: true} {
		t.Run(fmt.Sprint(stance), func(t *testing.T) {
			g, enemy, owner := garrisonedAttackBackGame(t)
			setStanceBack(g, enemy, stance, true)
			g.Submit(owner, harness.OrderGarrison([]uint64{unitAt(g, owner, grunt, 1, 0)}, findWall(g, owner).InternalID))
			g.EndTurn()
			g.EndTurn()
			hitBack := false
			for _, a := range firstUnit(g, enemy).TickActions {
				hitBack = hitBack || a.Order.Source == "attack_back"
			}
			if u := firstUnit(g, enemy); u.CombatStats.Health >= float64(u.CombatStats.MaxHealth) {
				t.Fatal("the garrisoned wall never hit the enemy, so there was nothing to attack back")
			}
			if damaged := findWall(g, owner).CombatStats.Health < wallHealthFull; hitBack != attacksBack || damaged != attacksBack {
				t.Errorf("attack back is %v and wall damaged is %v, want both %v for stance %d", hitBack, damaged, attacksBack, stance)
			}
		})
	}
}

func requireWallBeforeUnitsInZone(t *testing.T) {
	g, enemy, owner := standoffIn(t, spaceWith(0, 0, flatTerrain, unit(grunt, 0)), wallZoneSpace(unit(worker, 1)))
	inside := unitAt(g, owner, worker, 1, 0)
	attackWall(g, enemy, owner, wallAttackOrders["attack zone"], 2)
	if findWall(g, owner).CombatStats.Health >= wallHealthFull {
		t.Errorf("wall health %v, want the wall hit first", findWall(g, owner).CombatStats.Health)
	}
	for _, u := range g.Self(owner).Units {
		if u.InternalID == inside && u.CombatStats.Health != float64(u.CombatStats.MaxHealth) {
			t.Errorf("unit inside the wall zone has health %v, want it untouched while the wall stands", u.CombatStats.Health)
		}
	}
}

func TestWallAttacksInClosedLayout(t *testing.T) {
	for name, order := range wallAttackOrders {
		t.Run(name, func(t *testing.T) {
			g, enemy, owner := standoff(t, "")
			requireMetAtBorder(t, g, owner, attackWall(g, enemy, owner, order, 2))
		})
	}
	for name, order := range map[string]func(uint64) wallOrder{"attack unit": attackUnit, "move to wall zone": moveToWallZone} {
		t.Run(name, func(t *testing.T) {
			g, enemy, owner := standoff(t, unit(worker, 1))
			requireParkedAtBorder(t, g, owner, attackWall(g, enemy, owner, order(unitAt(g, owner, worker, 1, 0)), 3))
		})
	}
	t.Run("stances", requireStancesAttackWall)
	t.Run("stand ground", requireStandGround)
	t.Run("attack back", requireAttackBackAtWall)
	t.Run("ranged unit", requireRangedNeedsNoMeeting)
	t.Run("unit inside the wall zone", requireWallBeforeUnitsInZone)
	t.Run("from inside the wall space", requireInsideEntry)
}

func requireStancesAttackWall(t *testing.T) {
	stances := map[defs.UnitStance]bool{defs.UnitStance_PASSIVE: false, defs.UnitStance_DEFENSIVE: true, defs.UnitStance_AGGRESSIVE: true}
	for stance, attacks := range stances {
		t.Run(fmt.Sprint(stance), func(t *testing.T) {
			g, enemy, owner := standoff(t, "")
			setStance(g, enemy, stance)
			g.EndTurn()
			g.EndTurn()
			if damaged := findWall(g, owner).CombatStats.Health < wallHealthFull; damaged != attacks {
				t.Errorf("wall damaged is %v, want %v for stance %d", damaged, attacks, stance)
			}
		})
	}
}

func requireInsideEntry(t *testing.T) {
	g := startGame(t, mapDoc("", wallSpace(1, 0, unit(grunt, 0))+","+spaceWith(3, -3, flatTerrain, unit(worker, 1))))
	u := attackWall(g, g.Human(0), g.Human(1), wallAttackOrders["attack building"], 1)
	if sunk, damaging := meetingActions(u); u.Zone != (harness.Coord{X: -1, Y: 0}) || sunk != 0 || damaging == 0 {
		t.Errorf("zone %v, half moves %d, damage %d, want it inside the wall zone attacking with no half move", u.Zone, sunk, damaging)
	}
}

func TestWallAttacksInOpenLayout(t *testing.T) {
	around := spaceWith(1, -1, flatTerrain, "")
	for name, order := range wallAttackOrders {
		t.Run(name, func(t *testing.T) {
			g, enemy, owner := standoff(t, "", around)
			requireMetAtBorder(t, g, owner, attackWall(g, enemy, owner, order, 2))
		})
	}
	t.Run("attack unit goes around", func(t *testing.T) {
		g, enemy, owner := standoff(t, unit(worker, 1), around)
		attackWall(g, enemy, owner, attackUnit(unitAt(g, owner, worker, 1, 0)), 4)
		if at := where(g, enemy); at != (harness.Coord{X: 1, Y: 0}) || findWall(g, owner).CombatStats.Health != wallHealthFull {
			t.Errorf("attacker at %v, want it walked around into (1,0) with the wall untouched", at)
		}
	})
}

func TestUnseenWallStillBlocksTheCrossing(t *testing.T) {
	catapultSight := defs.UnitConfigs[grunt]
	catapultSight.Vision = defs.RisqVision{Space: defs.VisibilityGood, Edge_adjacent: defs.VisibilityPoor}
	defs.UnitConfigs[grunt] = catapultSight
	highWall := fmt.Sprintf(`{"x":1,"y":0,"terrain":%d,"zones":[{"x":-1,"y":0,"building":{"id":%d,"player":1}}]}`, mountainousTerrain, wall)
	g := startGame(t, mapDoc("", spaceWith(0, 0, flatTerrain, unit(grunt, 0))+","+highWall+","+spaceWith(3, -3, flatTerrain, unit(worker, 1))))
	enemy := g.Human(0)
	g.Submit(enemy, harness.OrderMove([]uint64{firstUnit(g, enemy).InternalID}, 1, 0))
	g.EndTurn()
	g.EndTurn()
	u := firstUnit(g, enemy)
	if u.Space != origin || u.Zone != borderZone || len(u.ActiveOrders) != 0 {
		t.Errorf("unit at space %v zone %v with %d orders, want it stopped at the border zone %v by the wall it touches", u.Space, u.Zone, len(u.ActiveOrders), borderZone)
	}
}

func TestBorderCrossingTurnsFollowTerrainCost(t *testing.T) {
	cases := map[string]struct {
		terrain, turns int
		why            string
	}{
		"flat":  {flatTerrain, 1, "a worker has 8 stamina and the crossing costs 1 + 6"},
		"hilly": {hillyTerrain, 2, "the 9-stamina crossing is only affordable with carried-over stamina"},
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
	g, mover := moverGame(t, spaceWith(1, 0, waterTerrain, ""), spaceWith(1, -1, flatTerrain, ""), spaceWith(2, -1, flatTerrain, ""), spaceWith(2, 0, flatTerrain, ""))
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
	g, mover := moverGame(t, spaceWith(2, 2, flatTerrain, ""), spaceWith(1, 0, flatTerrain, ""))
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

func TestMoveToImpassableSpaceStopsAtClosestReachableSpace(t *testing.T) {
	g, mover := moverGame(t, spaceWith(1, 0, flatTerrain, ""), spaceWith(2, 0, waterTerrain, ""))
	g.Submit(mover, harness.OrderMove([]uint64{firstUnit(g, mover).InternalID}, 2, 0))
	for turn := 0; turn < 4; turn++ {
		g.EndTurn()
	}
	if at := where(g, mover); at != (harness.Coord{X: 1, Y: 0}) {
		t.Errorf("unit at %v, want it stopped at (1,0) next to the water", at)
	}
	if zone := firstUnit(g, mover).Zone; zone != (harness.Coord{X: 1, Y: 0}) {
		t.Errorf("unit in zone %v, want the zone of (1,0) facing the water", zone)
	}
	if orders := g.Self(mover).ActiveOrders; len(orders) != 0 {
		t.Errorf("%d orders still queued, want the order finished at the closest point", len(orders))
	}
}

func TestMovesTheEngineCannotMakeAreDropped(t *testing.T) {
	cases := map[string]struct {
		target  harness.Coord
		terrain int
	}{
		"impassable":   {harness.Coord{X: 1, Y: 0}, waterTerrain},
		"disconnected": {harness.Coord{X: 2, Y: 2}, flatTerrain},
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

func TestEnemyReplayActionsAreRedacted(t *testing.T) {
	all := spaceWith(0, 0, flatTerrain, unit(worker, 0)+","+unit(worker, 1)) + "," + spaceWith(1, 0, flatTerrain, "")
	g := startGame(t, mapDoc("", all))
	p, enemy := g.Human(0), g.Human(1)
	mover := firstUnit(g, enemy)
	g.Submit(enemy, harness.OrderMove([]uint64{mover.InternalID}, 1, 0))
	g.EndTurn()
	seen := harness.RequireReplay(t, g.State(p)).Unit(mover.InternalID)
	if seen == nil || len(seen.TickActions) == 0 {
		t.Fatal("visible enemy actions missing from the replay")
	}
	for _, a := range seen.TickActions {
		if a.Order.Source != "" || a.Order.TargetID != 0 || a.Intent.MinCost != 0 || a.Intent.MaxCost != 0 || a.Intent.AvailableStamina != 0 || a.Execute.StaminaSpent != 0 {
			t.Errorf("enemy action leaks private data: %+v", a)
		}
	}
}

func TestReplayBaselineCarriesStartOfTurnVisibility(t *testing.T) {
	g, p := moverGame(t, spaceWith(1, 0, flatTerrain, ""))
	g.EndTurn()
	baseline := harness.RequireReplay(t, g.State(p)).Baseline
	found := false
	for _, s := range baseline.Spaces {
		if s.CoordinateKey == harness.SpaceKey(0, 0) {
			found = s.Visibility >= 3
		}
	}
	if !found {
		t.Errorf("baseline lacks good visibility for the player's own space: %+v", baseline.Spaces)
	}
}
