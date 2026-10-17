package shape

import (
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestFogWireShapeUsesFrozenContents(t *testing.T) {
	g := shapeGame(t, 4)
	queueWork(g)
	leaveTarget(g)
	reader, owner := g.Human(0), g.Human(1)
	remembered := space(t, snapshot(g, reader, false), 0, 0)
	equal(t, remembered["visibility"], float64(1))
	checkShape(t, remembered, exploredShape)
	checkFogContents(t, remembered)
	for _, value := range array(t, remembered["buildings"]) {
		building := checkShape(t, value, cachedBuildingShape)
		checkEntityDetails(t, building, "produces")
		for _, key := range []string{"turn_stamina", "current_stamina", "max_stamina"} {
			equal(t, building[key], float64(0))
		}
		equal(t, len(array(t, building["active_orders"])), 0)
		equal(t, len(array(t, building["production_queue"])), 0)
	}
	var deletes []defs.OrderFromFrontend
	for _, building := range g.Self(owner).Buildings {
		if building.Space == (harness.Coord{}) {
			deletes = append(deletes, harness.Order(defs.OrderType_BuildingDelete, []uint64{building.InternalID}, 0, false))
		}
	}
	deletes = append(deletes, harness.Order(defs.OrderType_UnitDelete, []uint64{unitID(g, 1, 0, 101)}, 0, false))
	g.Submit(owner, deletes...)
	g.EndTurn()
	g.EndTurn()
	live := space(t, snapshot(g, owner, false), 0, 0)
	equal(t, len(array(t, live["buildings"])), 0)
	equal(t, live["ownership"], float64(-1))
	frozen := space(t, snapshot(g, reader, false), 0, 0)
	checkShape(t, frozen, exploredShape)
	equal(t, frozen["ownership"], remembered["ownership"])
	equal(t, frozen["zones"], remembered["zones"])
	equal(t, frozen["resources"], remembered["resources"])
	for _, value := range array(t, remembered["buildings"]) {
		building := object(t, value)
		equal(t, entity(t, frozen, "buildings", uint64(building["internal_id"].(float64))), building)
	}
	previous := object(t, array(t, remembered["resources"])[0])["resources_left"].(float64)
	current := object(t, array(t, live["resources"])[0])["resources_left"].(float64)
	if current >= previous {
		t.Fatalf("live resource %v did not decrease from cached %v", current, previous)
	}
	enemy := player(t, snapshot(g, reader, false), 1)
	equal(t, len(array(t, enemy["units"])), 0)
	equal(t, len(array(t, enemy["buildings"])), 0)
	id := unitID(g, 0, -3, 15)
	g.Submit(reader, harness.OrderMove([]uint64{id}, -1, 0))
	for turn := 0; g.State(reader).Unit(id).Space != (harness.Coord{X: -1}); turn++ {
		if turn == 8 {
			t.Fatal("scout never restored vision")
		}
		g.EndTurn()
	}
	refreshed := space(t, snapshot(g, reader, false), 0, 0)
	equal(t, refreshed["visibility"], float64(4))
	equal(t, len(array(t, refreshed["buildings"])), 0)
	equal(t, refreshed["ownership"], float64(-1))
	equal(t, refreshed["resources"], space(t, snapshot(g, owner, false), 0, 0)["resources"])
}

func leaveTarget(g *harness.Game) {
	g.T.Helper()
	human, id := g.Human(0), unitID(g, 0, -1, 15)
	g.Submit(human, harness.OrderMove([]uint64{id}, -3, 0))
	for turn := 0; turn < 8; turn++ {
		g.EndTurn()
		if g.State(human).Unit(id).Space == (harness.Coord{X: -3}) {
			return
		}
	}
	g.T.Fatal("scout never left target vision")
}

const cachedBuildingShape = "internal_id:n player_id:n building_id:n display_name:s population_support:n combat_stats:o under_construction:b stamina_remaining:n construction_stamina_total:n turn_stamina:n current_stamina:n max_stamina:n produces:a active_orders:a production_queue:a zone_coordinate:o space_coordinate:o"

func checkFogContents(t *testing.T, target map[string]any) {
	t.Helper()
	for _, value := range array(t, target["resources"]) {
		resource := checkShape(t, value, resourceShape)
		checkShape(t, resource["space_coordinate"], coordinateShape)
		checkShape(t, resource["zone_coordinate"], coordinateShape)
	}
	for _, row := range array(t, target["zones"]) {
		for _, value := range array(t, row) {
			zone := object(t, value)
			checkShape(t, zone, targetZoneContract(t, zone))
			if zone["resource"] != nil {
				checkShape(t, zone["resource"], resourceShape)
			}
			if zone["building"] != nil {
				checkShape(t, zone["building"], cachedBuildingShape)
			}
		}
	}
}

func TestVisibilityWireShapes(t *testing.T) {
	for _, level := range []uint8{0, 2, 3, 4} {
		t.Run(fmt.Sprint(level), func(t *testing.T) {
			g := shapeGame(t, level)
			state := snapshot(g, g.Human(0), false)
			target := space(t, state, 0, 0)
			if target["visibility"] != float64(level) {
				t.Fatalf("visibility %v, want %d", target["visibility"], level)
			}
			if level == 0 {
				checkShape(t, target, unexploredShape)
				return
			}
			contract, units := exploredShape+" unit_count:n", "unit_count:n"
			if level >= 3 {
				contract, units = exploredShape+" units:a", "units:a"
			} else {
				equal(t, target["unit_count"], float64(2))
			}
			checkShape(t, target, contract)
			checkShape(t, target["coordinate"], coordinateShape)
			for _, row := range array(t, target["zones"]) {
				for _, value := range array(t, row) {
					zone := object(t, value)
					contract := targetZoneContract(t, zone) + " " + units
					checkShape(t, zone, contract)
					checkShape(t, zone["coordinate"], coordinateShape)
				}
			}
		})
	}
}

const coordinateShape = "x:n y:n"
const costShape = "food:n wood:n stone:n gold:n"
const identityShape = "client_id:n ai_player_id:n player_id:n nickname:s connected:b"
const gameShape = "board_size:n population_limit:n turn_number:n giving_orders:b background_image:s background_top_left:a background_top_right:a rotation:n has_economy:b has_buildings:b mercenaries_need_region:b game_base:o players:a spaces:a space_links:a regions:a last_turn_replay:o?"
const playerShape = "player:o population_limit:n max_population_limit:n unlimited_population:b score:n color:s orders_submitted:b eliminated:b buildings:a units:a active_orders:a researched_techs:o"
const privatePlayerShape = "resources:o turn_report:o? planned_foundations:a available_mercenaries:a auto_renewals:a"
const unexploredShape = "coordinate:o coordinate_key:n visibility:n"
const exploredShape = unexploredShape + " terrain_id:n terrain_type:n terrain_stats:o display_name:s gold_income:n ownership:n zones:a resources:a buildings:a"
const zoneShape = "coordinate:o coordinate_key:n terrain_override:n ownership:n"
const resourceShape = "internal_id:n resource_id:n display_name:s space_coordinate:o zone_coordinate:o resources_left:n max_resources:n base_gather_speed:n gather_capacity:n"
const unitShape = "internal_id:n player_id:n unit_id:n unit_type:n display_name:s space_coordinate:o zone_coordinate:o turn_stamina:n current_stamina:n max_stamina:n combat_stats:o attack_range:n active_orders:a builds:a tick_actions:a"
const privateUnitShape = "stance:n interrupt_current:b attack_back:b target_priority:a"
const buildingShape = "internal_id:n player_id:n building_id:n display_name:s space_coordinate:o zone_coordinate:o population_support:n combat_stats:o under_construction:b stamina_remaining:n construction_stamina_total:n turn_stamina:n current_stamina:n max_stamina:n garrison_capacity:n attack_range:n has_garrisoned_units:b produces:a active_orders:a production_queue:a tick_actions:a"
const privateBuildingShape = "garrisoned_units:a renewing:b auto_attack:b interrupt_current:b target_priority:a"
const farmShape = "resources_left:n resource_capacity:n renew_stamina:n gather_capacity:n base_gather_speed:n renew_cost:o resource_category:n"
const combatShape = "health:n max_health:n attack_type:n attack_blunt:n attack_piercing:n attack_magic:n defense_blunt:n defense_piercing:n defense_magic:n penetration_blunt:n penetration_piercing:n penetration_magic:n"
const orderShape = "internal_id:n player_id:n order_type:n target_id:n turn_received:n turn_resolved:n clear_previous_orders:b subjects:a"
const producibleShape = "row:n col:n kind:n id:n cost:o stamina_cost:n display_name:s description:s required_tech_id:n"
const garrisonUnitShape = "internal_id:n player_id:n unit_id:n unit_type:n display_name:s turn_stamina:n current_stamina:n max_stamina:n combat_stats:o attack_range:n active_orders:a builds:a garrisoned_in:n tick_actions:a"

func TestEnemyPrivateDataVisibility(t *testing.T) {
	for _, level := range []uint8{0, 2, 3, 4} {
		t.Run(fmt.Sprint(level), func(t *testing.T) {
			g := shapeGame(t, level)
			unit, center := queueWork(g)
			state := snapshot(g, g.Human(0), false)
			enemy := checkShape(t, player(t, state, 1), playerShape)
			equal(t, len(array(t, enemy["active_orders"])), 0)
			if level >= 2 {
				building := entity(t, enemy, "buildings", center)
				contract, count := buildingShape, 0
				if level == 4 {
					contract, count = contract+" "+privateBuildingShape, 1
				}
				checkShape(t, building, contract)
				equal(t, len(array(t, building["active_orders"])), count)
				equal(t, len(array(t, building["production_queue"])), count)
			} else {
				equal(t, len(array(t, enemy["buildings"])), 0)
			}
			if level >= 3 {
				u := entity(t, enemy, "units", unit)
				contract, count := unitShape, 0
				if level == 4 {
					contract, count = contract+" "+privateUnitShape, 1
					if u["move_path"] != nil {
						contract += " move_path:a"
					}
				}
				checkShape(t, u, contract)
				equal(t, len(array(t, u["active_orders"])), count)
			} else {
				equal(t, len(array(t, enemy["units"])), 0)
			}
			checkShape(t, space(t, state, 3, 0), unexploredShape)
		})
	}
}

func TestEnemyMovePathRequiresSpyVision(t *testing.T) {
	for _, level := range []uint8{3, 4} {
		t.Run(fmt.Sprint(level), func(t *testing.T) {
			g := configuredGame(t, level, func() {
				unit := defs.UnitConfigs[1]
				unit.Turn_stamina = 10
				defs.UnitConfigs[1] = unit
				scout := defs.UnitConfigs[15]
				scout.Vision.Secondary = level
				defs.UnitConfigs[15] = scout
			})
			id := unitID(g, 1, 0, 1)
			g.Submit(g.Human(1), harness.OrderMove([]uint64{id}, 3, 0))
			g.EndTurn()
			owner := player(t, snapshot(g, g.Human(1), false), 1)
			path := array(t, entity(t, owner, "units", id)["move_path"])
			if len(path) == 0 {
				t.Fatal("owner has no path to test privacy")
			}
			enemy := player(t, snapshot(g, g.Human(0), false), 1)
			u := entity(t, enemy, "units", id)
			if level == 4 {
				equal(t, u["move_path"], path)
			} else if _, exists := u["move_path"]; exists {
				t.Error("good visibility exposed enemy move path")
			}
		})
	}
}

func TestEntityCopiesAgreeAcrossSnapshot(t *testing.T) {
	for _, level := range []uint8{2, 3, 4} {
		t.Run(fmt.Sprint(level), func(t *testing.T) {
			g := shapeGame(t, level)
			queueWork(g)
			for _, human := range []int{g.Human(0), g.Human(1)} {
				state := snapshot(g, human, false)
				target := space(t, state, 0, 0)
				checkWorldCopies(t, state, target)
			}
		})
	}
}

func checkWorldCopies(t *testing.T, state, target map[string]any) {
	t.Helper()
	rows := array(t, target["zones"])
	equal(t, len(rows), 3)
	unitCount := 0
	for j, value := range rows {
		row := array(t, value)
		equal(t, len(row), []int{2, 3, 2}[j])
		for _, value := range row {
			zone := object(t, value)
			for _, kind := range []string{"building", "resource"} {
				if zone[kind] != nil {
					item := object(t, zone[kind])
					id := uint64(item["internal_id"].(float64))
					equal(t, entity(t, target, kind+"s", id), item)
					if kind == "building" {
						owner := player(t, state, int(item["player_id"].(float64)))
						equal(t, entity(t, owner, "buildings", id), item)
					} else {
						checkShape(t, item, resourceShape)
					}
				}
			}
			if zone["units"] != nil {
				units := array(t, zone["units"])
				unitCount += len(units)
				for _, value := range units {
					unit := object(t, value)
					id := uint64(unit["internal_id"].(float64))
					owner := player(t, state, int(unit["player_id"].(float64)))
					equal(t, entity(t, target, "units", id), unit)
					equal(t, entity(t, owner, "units", id), unit)
				}
			} else {
				unitCount += int(zone["unit_count"].(float64))
			}
		}
	}
	if target["units"] != nil {
		equal(t, len(array(t, target["units"])), unitCount)
	} else {
		equal(t, target["unit_count"], float64(unitCount))
	}
}
