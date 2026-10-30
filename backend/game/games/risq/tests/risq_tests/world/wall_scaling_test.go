package world

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

const escaladeTech = 308

func fogWallGame(t *testing.T, stamina int, building_id uint32) *harness.Game {
	t.Helper()
	config := defs.UnitConfigs[worker]
	config.Vision = defs.RisqVision{Space: defs.VisibilityGood}
	config.Turn_stamina = stamina
	defs.UnitConfigs[worker] = config
	t.Cleanup(func() { pristine.Restore(baseDir) })
	old := strings.Replace(wallZoneSpace(unit(worker, 1)), `"id":211`, fmt.Sprintf(`"id":%d`, building_id), 1)
	spaces := spaceWith(0, 0, flatTerrain, unit(worker, 0)) + "," + old + "," + spaceWith(2, 0, flatTerrain, unit(grunt, 1))
	writeCustomMap("fog-wall", mapDoc(`,"starting_techs":[308]`, spaces))
	return harness.NewGameWithSettings(t, "custom:fog-wall", 1, 2, map[string]any{"visibility": float64(defs.VisibilityMode_EXPLORED)})
}

func TestFoggedReplacementWallChargesScalingCost(t *testing.T) {
	g := fogWallGame(t, 10, shelter)
	mover, owner := g.Human(0), g.Human(1)
	g.Submit(owner, harness.Order(defs.OrderType_BuildingDelete, []uint64{g.Self(owner).Buildings[0].InternalID}, 0, false))
	g.EndTurn()
	g.Submit(owner, harness.OrderBuild([]uint64{unitAt(g, owner, worker, 1, 0)}, wall, 1, 0, -1, 0))
	g.EndTurn()
	known := g.State(mover).Space(1, 0)
	if known.Visibility != fog || len(known.Buildings) != 1 || known.Buildings[0].BuildingID != shelter || findWall(g, owner).UnderConstruction {
		t.Fatal("fixture must remember a shelter while a completed wall occupies the fogged zone")
	}
	g.Submit(mover, harness.OrderMoveZone([]uint64{firstUnit(g, mover).InternalID}, 1, 0, -1, 0))
	g.EndTurn()
	if costs := executedMoveCosts(g, mover); !slices.Equal(costs, []int{1, 9}) {
		t.Fatalf("movement costs %v, want [1 9] including the live wall surcharge", costs)
	}
}

func TestFoggedWallRequiresActualClimbingStamina(t *testing.T) {
	g := fogWallGame(t, 5, shelter)
	mover, owner := g.Human(0), g.Human(1)
	g.Submit(owner, harness.Order(defs.OrderType_BuildingDelete, []uint64{g.Self(owner).Buildings[0].InternalID}, 0, false))
	g.EndTurn()
	g.Submit(owner, harness.OrderBuild([]uint64{unitAt(g, owner, worker, 1, 0)}, wall, 1, 0, -1, 0))
	g.EndTurn()
	g.EndTurn()
	if findWall(g, owner).UnderConstruction || g.State(mover).Space(1, 0).Buildings[0].BuildingID != shelter {
		t.Fatal("fixture must contain a completed wall remembered as a shelter")
	}
	g.Submit(mover, harness.OrderMoveZone([]uint64{firstUnit(g, mover).InternalID}, 1, 0, -1, 0))
	g.EndTurn()
	at := firstUnit(g, mover)
	blocked := harness.RequireTickBlocked(t, at.TickActions, "insufficient_stamina")
	if at.Space != origin || at.Zone != borderZone || len(at.ActiveOrders) == 0 || blocked.Intent.MinCost != 9 || blocked.Execute.StaminaSpent != 0 {
		t.Fatalf("climber must wait at the border for nine stamina: %+v", at)
	}
}

func TestDemolishedFoggedWallDoesNotChargeClimbingStamina(t *testing.T) {
	g := fogWallGame(t, 5, wall)
	mover, owner := g.Human(0), g.Human(1)
	g.Submit(owner, harness.Order(defs.OrderType_BuildingDelete, []uint64{findWall(g, owner).InternalID}, 0, false))
	g.EndTurn()
	known := g.State(mover).Space(1, 0)
	if known.Visibility != fog || len(known.Buildings) != 1 || known.Buildings[0].BuildingID != wall || len(g.Self(owner).Buildings) != 0 {
		t.Fatal("fixture must remember a demolished wall")
	}
	g.Submit(mover, harness.OrderMoveZone([]uint64{firstUnit(g, mover).InternalID}, 1, 0, -1, 0))
	g.EndTurn()
	if at := firstUnit(g, mover); at.Space != (harness.Coord{X: 1}) || !slices.Equal(executedMoveCosts(g, mover), []int{1, 6}) {
		t.Fatalf("climber must cross for six stamina despite the remembered wall: %+v", at)
	}
}

func TestFoggedGarrisonBlocksClimbingWithoutSpendingStamina(t *testing.T) {
	config := defs.BuildingConfigs[wall]
	config.Garrison_capacity = 5
	defs.BuildingConfigs[wall] = config
	g := fogWallGame(t, 10, wall)
	mover, owner := g.Human(0), g.Human(1)
	defender := unitAt(g, owner, worker, 1, 0)
	g.Submit(owner, harness.OrderGarrison([]uint64{defender}, findWall(g, owner).InternalID))
	g.EndTurn()
	if g.State(mover).Space(1, 0).Visibility != fog || g.State(owner).Unit(defender).GarrisonedIn == nil {
		t.Fatal("fixture must hide the wall's new garrison")
	}
	g.Submit(mover, harness.OrderMoveZone([]uint64{firstUnit(g, mover).InternalID}, 1, 0, -1, 0))
	g.EndTurn()
	at := firstUnit(g, mover)
	blocked := harness.RequireTickBlocked(t, at.TickActions, "blocked_by_wall")
	if at.Space != origin || at.Zone != borderZone || blocked.Intent.MinCost != 6 || !slices.Equal(executedMoveCosts(g, mover), []int{1}) {
		t.Fatalf("blocked climbing must use plain cost and spend nothing: %+v", at)
	}
}

func TestGarrisonedClimberUngarrisonsBeforeMoving(t *testing.T) {
	g := wallScalingGame(t, `,"starting_techs":[308]`)
	owner := g.Human(1)
	id := firstUnit(g, owner).InternalID
	g.Submit(owner, harness.OrderGarrison([]uint64{id}, findWall(g, owner).InternalID))
	g.EndTurn()
	if firstUnit(g, owner).GarrisonedIn == nil {
		t.Fatal("fixture did not garrison the climber")
	}
	g.Submit(owner, harness.OrderMoveZone([]uint64{id}, 1, 0, 0, 0))
	g.EndTurn()
	at := firstUnit(g, owner)
	action := harness.RequireTickAction(t, at.TickActions, "move")
	if at.GarrisonedIn != nil || at.Zone != origin || action.Intent.Resolution.Reason != "automatic_ungarrison" {
		t.Fatalf("climber must ungarrison before resolving movement: %+v", at)
	}
}

func TestFoggedWallBorderCombatUsesPlainMovementCost(t *testing.T) {
	config := defs.UnitConfigs[worker]
	config.Attack_type, config.Attack_blunt = defs.AttackType_BLUNT, 1
	defs.UnitConfigs[worker] = config
	g := fogWallGame(t, 10, wall)
	mover, owner := g.Human(0), g.Human(1)
	g.Action(owner, "set-unit-behavior", gin.H{"internal_ids": []uint64{unitAt(g, owner, worker, 1, 0)}, "stance": uint8(defs.UnitStance_PASSIVE)})
	g.Submit(mover, wallAttackOrders["attack zone"](findWall(g, owner).InternalID, firstUnit(g, mover).InternalID))
	g.EndTurn()
	at := firstUnit(g, mover)
	sunk, damaging := meetingActions(at)
	if at.Space != origin || at.Zone != borderZone || sunk != 1 || damaging == 0 {
		t.Fatalf("climber must attack the fogged wall at the border: %+v", at)
	}
	for _, action := range at.TickActions {
		if paid := action.Intent.Resolution.SunkCost; paid != 0 && paid != 3 {
			t.Fatalf("border half move cost %d, want three without climbing surcharge", paid)
		}
	}
}

func TestSameTickGarrisonBlocksWallScaling(t *testing.T) {
	g := wallScalingGame(t, `,"starting_techs":[308]`)
	enemy, owner := g.Human(0), g.Human(1)
	climber, defender := firstUnit(g, enemy), firstUnit(g, owner)
	g.Submit(enemy, harness.OrderMoveZone([]uint64{climber.InternalID}, 0, 0, 1, 0))
	g.EndTurn()
	g.Submit(owner, harness.OrderGarrison([]uint64{defender.InternalID}, findWall(g, owner).InternalID))
	g.Submit(enemy, harness.OrderMoveZone([]uint64{climber.InternalID}, 1, 0, -1, 0))
	g.EndTurn()
	if firstUnit(g, owner).GarrisonedIn == nil {
		t.Fatal("defender did not garrison")
	}
	if at := firstUnit(g, enemy); at.Space != origin || at.Zone != borderZone {
		t.Fatalf("climber crossed a wall receiving a garrison: %+v", at)
	}
}

// P0's worker stands in (0,0) facing P1's wall at (1,0), with a P1 worker inside the wall's zone
func wallScalingGame(t *testing.T, startingTechs string) *harness.Game {
	t.Helper()
	config := defs.BuildingConfigs[wall]
	config.Garrison_capacity = 5
	defs.BuildingConfigs[wall] = config
	spaces := spaceWith(0, 0, flatTerrain, unit(worker, 0)) + "," + wallZoneSpace(unit(worker, 1))
	return startGame(t, mapDoc(startingTechs, spaces))
}

func TestWallScalingTechLetsUnitsEnterTheWallZone(t *testing.T) {
	cases := map[string]struct {
		techs  string
		arrive int
		costs  []int
	}{
		"without-tech": {"", -1, nil},
		"with-tech":    {`,"starting_techs":[308]`, 1, []int{1, 9}},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			g := wallScalingGame(t, c.techs)
			enemy := g.Human(0)
			if turns := moveUntilArrived(g, enemy, 1, 0, 4); turns != c.arrive {
				t.Fatalf("arrived after %d turns, want %d", turns, c.arrive)
			}
			if c.costs != nil && !slices.Equal(executedMoveCosts(g, enemy), c.costs) {
				t.Errorf("move steps cost %v, want %v", executedMoveCosts(g, enemy), c.costs)
			}
		})
	}
}

func TestAttackOrdersOnWallIgnoreScaling(t *testing.T) {
	for name, order := range wallAttackOrders {
		t.Run(name, func(t *testing.T) {
			config := defs.UnitConfigs[worker]
			config.Attack_type, config.Attack_blunt = defs.AttackType_BLUNT, 1
			defs.UnitConfigs[worker] = config
			g := wallScalingGame(t, `,"starting_techs":[308]`)
			enemy, owner := g.Human(0), g.Human(1)
			requireMetAtBorder(t, g, owner, attackWall(g, enemy, owner, order, 2))
		})
	}
}

func TestGarrisonedWallCannotBeScaled(t *testing.T) {
	g := wallScalingGame(t, `,"starting_techs":[308]`)
	enemy, owner := g.Human(0), g.Human(1)
	defender := firstUnit(g, owner)
	g.Submit(owner, harness.OrderGarrison([]uint64{defender.InternalID}, findWall(g, owner).InternalID))
	g.EndTurn()
	if firstUnit(g, owner).GarrisonedIn == nil {
		t.Fatal("fixture did not garrison the defender in the wall")
	}
	if turns := moveUntilArrived(g, enemy, 1, 0, 4); turns != -1 {
		t.Errorf("scaling unit arrived after %d turns, want the garrisoned wall to block it", turns)
	}
}

func TestRangedClimberCanAttackZoneThroughTheWallZone(t *testing.T) {
	archer := defs.UnitConfigs[grunt]
	archer.Attack_range, archer.Vision.Adjacent = defs.RisqRange_WALL, defs.VisibilityGood
	defs.UnitConfigs[grunt] = archer
	tech := defs.TechConfigs[escaladeTech]
	tech.Affects_unit_ids = []uint32{grunt}
	defs.TechConfigs[escaladeTech] = tech
	g := startGame(t, mapDoc(`,"starting_techs":[308]`, borderEnemySpace()+","+wallSpace(1, 0, unit(worker, 1))))
	enemy, owner := g.Human(0), g.Human(1)
	setStance(g, owner, defs.UnitStance_PASSIVE)
	g.Submit(enemy, harness.Order(defs.OrderType_UnitAttackZone, []uint64{firstUnit(g, enemy).InternalID}, harness.ZoneKey(1, 0, 0, 0), false))
	g.EndTurn()
	g.EndTurn()
	if at := firstUnit(g, enemy); at.Space != (harness.Coord{X: 1}) {
		t.Errorf("ranged climber at space %v, want it to have climbed into the wall space to attack the center zone", at.Space)
	}
}

func TestRangedClimberCanApproachThroughTheWallZone(t *testing.T) {
	archer := defs.UnitConfigs[grunt]
	archer.Attack_range = defs.RisqRange_WALL
	archer.Vision.Adjacent = defs.VisibilityGood
	defs.UnitConfigs[grunt] = archer
	tech := defs.TechConfigs[escaladeTech]
	tech.Affects_unit_ids = []uint32{grunt}
	defs.TechConfigs[escaladeTech] = tech
	g := startGame(t, mapDoc(`,"starting_techs":[308]`, borderEnemySpace()+","+wallSpace(1, 0, unit(worker, 1))))
	enemy, owner := g.Human(0), g.Human(1)
	setStance(g, owner, defs.UnitStance_PASSIVE)
	g.Submit(enemy, harness.OrderAttackUnit([]uint64{firstUnit(g, enemy).InternalID}, firstUnit(g, owner).InternalID))
	g.EndTurn()
	g.EndTurn()
	if at := firstUnit(g, enemy); at.Space != (harness.Coord{X: 1}) {
		t.Errorf("ranged climber at space %v, want it to have climbed into the wall space to reach its target", at.Space)
	}
}
