package shape

import (
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

func queueWork(g *harness.Game) (uint64, uint64) {
	g.T.Helper()
	unit, center := unitID(g, 1, 0, 11), centerID(g)
	produce := harness.OrderProduce([]uint64{center}, 11)
	g.Submit(g.Human(1), harness.OrderGather([]uint64{unit}, 0, 0, 1, 0), produce, produce)
	g.EndTurn()
	return unit, center
}

func TestOrderAndProductionQueueWireShapes(t *testing.T) {
	g := shapeGame(t, 4)
	unit, center := queueWork(g)
	owner := player(t, snapshot(g, g.Human(1), false), 1)
	orders := array(t, owner["active_orders"])
	equal(t, len(orders), 2)
	for _, value := range orders {
		order := checkShape(t, value, orderShape)
		equal(t, order["player_id"], float64(1))
		equal(t, len(array(t, order["subjects"])), 1)
	}
	u := entity(t, owner, "units", unit)
	uOrders := array(t, u["active_orders"])
	equal(t, len(uOrders), 1)
	uOrder := checkShape(t, uOrders[0], orderShape)
	equal(t, uOrder["subjects"], []any{float64(unit)})
	equal(t, uOrder["target_id"], float64(harness.ZoneKey(0, 0, 1, 0)))
	equal(t, uOrder["clear_previous_orders"], false)
	b := entity(t, owner, "buildings", center)
	queue := array(t, b["production_queue"])
	equal(t, len(queue), 1)
	item := checkShape(t, queue[0], "kind:n item_id:n stamina_remaining:n order_internal_id:n")
	bOrders := array(t, b["active_orders"])
	equal(t, len(bOrders), 1)
	order := checkShape(t, bOrders[0], orderShape)
	equal(t, order["subjects"], []any{float64(center)})
	equal(t, order["target_id"], float64(11))
	equal(t, item["order_internal_id"], order["internal_id"])
	equal(t, item["kind"], float64(1))
	equal(t, item["item_id"], float64(11))
}

func TestChangedBehaviorWireValues(t *testing.T) {
	g := configuredGame(t, 4, func() {
		center := defs.BuildingConfigs[201]
		center.Attack_type = defs.AttackType_BLUNT
		defs.BuildingConfigs[201] = center
	})
	owner, unit, center := g.Human(1), unitID(g, 1, 0, 21), centerID(g)
	g.Action(owner, "set-unit-behavior", gin.H{"internal_ids": []uint64{unit}, "stance": 1, "interrupt_current": true, "attack_back": false, "target_priority": []uint8{3, 1, 2}})
	g.Action(owner, "set-building-behavior", gin.H{"internal_ids": []uint64{center}, "auto_attack": false, "interrupt_current": true, "target_priority": []uint8{2, 3, 1}})
	for human := range 2 {
		p := player(t, snapshot(g, human, false), 1)
		u := checkShape(t, entity(t, p, "units", unit), unitShape+" "+privateUnitShape)
		equal(t, u["stance"], float64(1))
		equal(t, u["interrupt_current"], true)
		equal(t, u["attack_back"], false)
		equal(t, u["target_priority"], []any{float64(3), float64(1), float64(2)})
		b := checkShape(t, entity(t, p, "buildings", center), buildingShape+" "+privateBuildingShape)
		equal(t, b["auto_attack"], false)
		equal(t, b["interrupt_current"], true)
		equal(t, b["target_priority"], []any{float64(2), float64(3), float64(1)})
	}
}

func TestMovePathWireShape(t *testing.T) {
	g := shapeGame(t, 3)
	id, human := unitID(g, 0, -1, 15), g.Human(0)
	g.Submit(human, harness.OrderMove([]uint64{id}, -4, 0))
	g.EndTurn()
	owner := player(t, snapshot(g, human, false), 0)
	u := entity(t, owner, "units", id)
	checkShape(t, u, unitShape+" "+privateUnitShape+" move_path:a")
	path := array(t, u["move_path"])
	if len(path) == 0 {
		t.Fatal("moving scout has no path")
	}
	for _, value := range path {
		step := checkShape(t, value, "space:o zone:o")
		checkShape(t, step["space"], coordinateShape)
		checkShape(t, step["zone"], coordinateShape)
	}
	last := object(t, path[len(path)-1])
	equal(t, object(t, last["space"])["x"], float64(-4))
}

func TestUnderConstructionBuildingWireShape(t *testing.T) {
	g := configuredGame(t, 4, func() {
		unit := defs.UnitConfigs[11]
		unit.Turn_stamina = 10
		defs.UnitConfigs[11] = unit
		shelter := defs.BuildingConfigs[202]
		shelter.Build_stamina = 90
		defs.BuildingConfigs[202] = shelter
	})
	g.Submit(g.Human(1), harness.OrderBuild([]uint64{unitID(g, 1, 0, 11)}, 202, 0, 0, 0, 1))
	g.EndTurn()
	for human := range 2 {
		owner := player(t, snapshot(g, human, false), 1)
		found := false
		for _, value := range array(t, owner["buildings"]) {
			building := object(t, value)
			if building["building_id"] == float64(202) {
				found = true
				checkShape(t, building, buildingShape+" "+privateBuildingShape)
				equal(t, building["under_construction"], true)
				equal(t, building["construction_stamina_total"], float64(90))
				if remaining := building["stamina_remaining"].(float64); remaining <= 0 || remaining >= 90 {
					t.Errorf("construction remaining %v, want partial progress", remaining)
				}
			}
		}
		if !found {
			t.Fatal("under-construction shelter missing")
		}
	}
}

func TestFoundationWireShapeAndPrivacy(t *testing.T) {
	g := shapeGame(t, 4)
	unit, _ := queueWork(g)
	owner := g.Human(1)
	g.Submit(owner, harness.OrderBuild([]uint64{unit}, 202, 1, 0, 0, 0))
	g.EndTurn()
	p := player(t, snapshot(g, owner, false), 1)
	foundations := array(t, p["planned_foundations"])
	equal(t, len(foundations), 1)
	foundation := checkShape(t, foundations[0], "coordinate_key:n building_id:n display_name:s stamina_cost:n cost:o")
	checkShape(t, foundation["cost"], costShape)
	equal(t, foundation["coordinate_key"], float64(harness.ZoneKey(1, 0, 0, 0)))
	equal(t, foundation["building_id"], float64(202))
	equal(t, object(t, foundation["cost"])["wood"], float64(30))
	checkShape(t, player(t, snapshot(g, g.Human(0), false), 1), playerShape)
}

func garrisonSetup(g *harness.Game) (uint64, uint64) {
	g.T.Helper()
	unit, center, owner := unitID(g, 1, 0, 11), centerID(g), g.Human(1)
	g.Action(owner, "set-gather-point", gin.H{"building_id": center, "location_kind": 1, "location_id": harness.SpaceKey(1, 0), "object_type": 0, "object_id": 0})
	g.Submit(owner, harness.OrderGarrison([]uint64{unit}, center))
	g.EndTurn()
	if g.State(owner).Unit(unit).GarrisonedIn == nil {
		g.T.Fatal("worker did not garrison")
	}
	return unit, center
}

func TestGarrisonAndGatherPointVisibility(t *testing.T) {
	for _, level := range []uint8{2, 3, 4} {
		t.Run(fmt.Sprint(level), func(t *testing.T) {
			g := shapeGame(t, level)
			unit, center := garrisonSetup(g)
			owner := player(t, snapshot(g, g.Human(1), false), 1)
			u := checkShape(t, entity(t, owner, "units", unit), garrisonUnitShape+" "+privateUnitShape)
			equal(t, u["garrisoned_in"], float64(center))
			building := entity(t, owner, "buildings", center)
			checkShape(t, building, buildingShape+" "+privateBuildingShape+" gather_point:o")
			checkShape(t, building["gather_point"], "location_kind:n location_id:n object_type:n object_id:n")
			equal(t, building["garrisoned_units"], []any{float64(unit)})
			enemy := player(t, snapshot(g, g.Human(0), false), 1)
			b := entity(t, enemy, "buildings", center)
			contract := buildingShape
			if level == 4 {
				contract += " " + privateBuildingShape + " gather_point:o"
				equal(t, b["garrisoned_units"], []any{float64(unit)})
				equal(t, b["gather_point"], building["gather_point"])
			}
			checkShape(t, b, contract)
			equal(t, b["has_garrisoned_units"], true)
			for _, value := range array(t, enemy["units"]) {
				if object(t, value)["internal_id"] == float64(unit) {
					t.Error("enemy player list exposed a garrisoned unit")
				}
			}
		})
	}
}

func TestRenewingOrchardWireShape(t *testing.T) {
	g := configuredGame(t, 4, func() {
		orchard := defs.BuildingConfigs[203]
		orchard.Gather.Starting_resources = 0
		defs.BuildingConfigs[203] = orchard
	})
	owner := player(t, snapshot(g, g.Human(1), false), 1)
	var orchardID uint64
	for _, value := range array(t, owner["buildings"]) {
		orchard := object(t, value)
		if orchard["building_id"] == float64(203) {
			orchardID = uint64(orchard["internal_id"].(float64))
		}
	}
	if orchardID == 0 {
		t.Fatal("orchard missing")
	}
	g.Submit(g.Human(1), harness.Order(defs.OrderType_UnitRenew, []uint64{unitID(g, 1, 0, 11)}, int64(orchardID), false))
	g.EndTurn()
	for human := range 2 {
		p := player(t, snapshot(g, human, false), 1)
		orchard := checkShape(t, entity(t, p, "buildings", orchardID), buildingShape+" "+privateBuildingShape+" "+orchardShape+" renew_stamina_remaining:n")
		checkShape(t, orchard["renew_cost"], costShape)
		equal(t, orchard["resources_left"], float64(0))
		equal(t, orchard["renewing"], true)
		equal(t, orchard["display_name"], "Test Orchard (expired)")
	}
}

func TestOwnedEntityWireShapes(t *testing.T) {
	g := shapeGame(t, 3)
	owner := player(t, snapshot(g, g.Human(1), false), 1)
	for _, value := range array(t, owner["units"]) {
		unit := checkShape(t, value, unitShape+" "+privateUnitShape)
		checkEntityDetails(t, unit, "builds")
	}
	for _, value := range array(t, owner["buildings"]) {
		building := object(t, value)
		contract := buildingShape + " " + privateBuildingShape
		if building["building_id"] == float64(203) {
			contract += " " + orchardShape
			checkShape(t, building["renew_cost"], costShape)
		}
		checkShape(t, building, contract)
		checkEntityDetails(t, building, "produces")
	}
	target := space(t, snapshot(g, g.Human(1), false), 0, 0)
	resources := array(t, target["resources"])
	equal(t, len(resources), 1)
	resource := checkShape(t, resources[0], resourceShape)
	checkShape(t, resource["space_coordinate"], coordinateShape)
	checkShape(t, resource["zone_coordinate"], coordinateShape)
	equal(t, resource["resource_id"], float64(41))
}

func TestBackgroundAndSpaceLinksWireShapes(t *testing.T) {
	saved15 := defs.UnitConfigs[15]
	t.Cleanup(func() { defs.UnitConfigs[15] = saved15 })
	defs.UnitConfigs[15] = defs.UnitConfigs[11]
	g := harness.NewGame(t, "custom:metadata", 1, 2)
	equal(t, len(array(t, snapshot(g, g.Human(0), true)["space_links"])), 0)
	state := checkShape(t, snapshot(g, g.Human(0), false), gameShape)
	equal(t, state["background_image"], "map.png")
	equal(t, state["background_top_left"], []any{float64(-4), float64(-2)})
	equal(t, state["background_top_right"], []any{float64(4), float64(2)})
	links := array(t, state["space_links"])
	equal(t, len(links), 1)
	link := checkShape(t, links[0], "from:o to:o direction:n to_direction:n")
	checkShape(t, link["from"], coordinateShape)
	checkShape(t, link["to"], coordinateShape)
	equal(t, link["from"], map[string]any{"x": float64(-4), "y": float64(0)})
	equal(t, link["to"], map[string]any{"x": float64(4), "y": float64(0)})
	equal(t, link["direction"], float64(3))
	equal(t, link["to_direction"], float64(0))
}

func checkEntityDetails(t *testing.T, entity map[string]any, actions string) {
	t.Helper()
	checkShape(t, entity["combat_stats"], combatShape)
	checkShape(t, entity["space_coordinate"], coordinateShape)
	checkShape(t, entity["zone_coordinate"], coordinateShape)
	for _, value := range array(t, entity[actions]) {
		checkProducible(t, value)
	}
}

func checkProducible(t *testing.T, value any) {
	t.Helper()
	item := object(t, value)
	contract := producibleShape
	switch item["kind"] {
	case float64(1):
		contract += " unit_type:n stats:o"
		checkShape(t, item["stats"], "health:n attack_type:n attack_blunt:n attack_piercing:n attack_range:n defense_blunt:n defense_piercing:n penetration_blunt:n penetration_piercing:n")
	case float64(2):
		contract += " affects_unit_ids:a affects_unit_types:a affects_buildings:a gather_bonus:o bonus:o unlocks_mercenaries:a additional_effects:a"
		checkShape(t, item["bonus"], "max_health:n turn_stamina:n range:n splash_damage_percent:n splash_max_targets:n reduce_inter_move_cost:n reduce_intra_move_cost:n scale_walls:b attack_blunt:n attack_piercing:n attack_magic:n defense_blunt:n defense_piercing:n defense_magic:n penetration_blunt:n penetration_piercing:n penetration_magic:n gather_capacity:n gather_speed:n resource_capacity:n renew_cost_discount:n garrison_capacity:n garrison_attack_cap_blunt:n garrison_attack_cap_piercing:n garrison_attack_cap_magic:n population_support:n")
		for _, unlocked := range array(t, item["unlocks_mercenaries"]) {
			checkShape(t, unlocked, "id:n display_name:s")
		}
	case float64(3):
		contract += " population_support:n garrison_capacity:n max_per_player:n is_wall:b"
	default:
		t.Fatalf("unknown producible kind %v", item["kind"])
	}
	checkShape(t, item, contract)
	checkShape(t, item["cost"], costShape)
}
