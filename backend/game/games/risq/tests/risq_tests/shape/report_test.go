package shape

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestTurnReportWireShape(t *testing.T) {
	g := shapeGame(t, 3)
	owner := g.Human(1)
	equal(t, player(t, snapshot(g, owner, false), 1)["turn_report"], nil)
	queueWork(g)
	p := player(t, snapshot(g, owner, false), 1)
	report := checkShape(t, p["turn_report"], "turn:n eliminated:b scores:a land:o regions:a resources:a population:o production:o combat:a orders:o")
	checkShape(t, report["land"], "held_start:n held_end:n gained:n lost:n newly_explored:n gold_from_land:n")
	checkShape(t, report["population"], "start:n end:n cap_start:n cap_end:n")
	production := checkShape(t, report["production"], "units_created:a units_hired:a buildings_built:a techs_researched:a")
	equal(t, len(array(t, production["units_hired"])), 0)
	created := array(t, production["units_created"])
	equal(t, len(created), 1)
	checkShape(t, created[0], "unit_id:n count:n")
	orders := checkShape(t, report["orders"], "active:n added:n failed:n executed:n cancelled:n failures:a")
	equal(t, len(array(t, orders["failures"])), 0)
	resources := array(t, report["resources"])
	equal(t, len(resources), 4)
	for i, value := range resources {
		line := checkShape(t, value, "category:n start:n gathered:n income:n spent:n final:n")
		equal(t, line["category"], float64(i+1))
	}
	for _, value := range array(t, report["scores"]) {
		checkShape(t, value, "player_id:n was:n now:n eliminated:b eliminated_this_turn:b")
	}
	for _, value := range array(t, report["regions"]) {
		checkShape(t, value, "name:s held_start:b held_end:b gold_bonus:n")
	}
	equal(t, len(array(t, production["buildings_built"])), 0)
	equal(t, len(array(t, production["techs_researched"])), 0)
	equal(t, len(array(t, report["combat"])), 0)
}

func TestReportRejectedOrderWireShape(t *testing.T) {
	g := configuredGame(t, 3, func() {
		unit := defs.UnitConfigs[11]
		unit.Cost.Food = 2000
		defs.UnitConfigs[11] = unit
	})
	owner := g.Human(1)
	order := harness.OrderProduce([]uint64{centerID(g)}, 11)
	g.Submit(owner, order)
	g.EndTurn()
	report := object(t, player(t, snapshot(g, owner, false), 1)["turn_report"])
	failures := array(t, object(t, report["orders"])["failures"])
	equal(t, len(failures), 1)
	failure := checkShape(t, failures[0], "order_type:n target_id:n reason:s")
	equal(t, failure["order_type"], float64(order.Order_type))
	equal(t, failure["target_id"], float64(order.Target_id))
	equal(t, failure["reason"], "cannot afford unit")
}

func TestCombatReportWireShapeForBothPlayers(t *testing.T) {
	g := configuredGame(t, 3, func() {
		unit := defs.UnitConfigs[21]
		unit.Attack_range, unit.Attack_blunt = defs.RisqRange_ADJACENT, 200
		defs.UnitConfigs[21] = unit
	})
	attacker, victim := unitID(g, 1, 0, 21), unitID(g, 0, -1, 15)
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

func TestReportSeparatesTrainedAndHiredUnits(t *testing.T) {
	g := configuredGame(t, 3, func() {
		center := defs.BuildingConfigs[201]
		center.Produces = []defs.Producible{{Kind: defs.ProducibleKind_UNIT, Id: 21}}
		defs.BuildingConfigs[201] = center
		grunt := defs.UnitConfigs[21]
		grunt.Production_stamina = 10
		defs.UnitConfigs[21] = grunt
	})
	owner := g.Human(1)
	g.Submit(owner, harness.OrderProduce([]uint64{centerID(g)}, 21), harness.Order(defs.OrderType_BuyMercenary, nil, harness.BuildKey(21, 0, 0, 0, 0), false))
	g.EndTurn()
	p := player(t, snapshot(g, owner, false), 1)
	production := object(t, object(t, p["turn_report"])["production"])
	for _, key := range []string{"units_created", "units_hired"} {
		entries := array(t, production[key])
		equal(t, len(entries), 1)
		equal(t, checkShape(t, entries[0], "unit_id:n count:n"), map[string]any{"unit_id": float64(21), "count": float64(1)})
	}
}

func TestReportBuildingAndResearchWireShapes(t *testing.T) {
	g := configuredGame(t, 3, func() {
		unit := defs.UnitConfigs[11]
		unit.Turn_stamina = 30
		defs.UnitConfigs[11] = unit
		shelter := defs.BuildingConfigs[202]
		shelter.Build_stamina = 1
		defs.BuildingConfigs[202] = shelter
		tech := defs.TechConfigs[301]
		tech.Research_stamina = 1
		defs.TechConfigs[301] = tech
	})
	owner := g.Human(1)
	build := harness.OrderBuild([]uint64{unitID(g, 1, 0, 11)}, 202, 0, 0, 0, 1)
	research := harness.Order(defs.OrderType_BuildingResearch, []uint64{centerID(g)}, 301, false)
	g.Submit(owner, build, research)
	g.EndTurn()
	p := player(t, snapshot(g, owner, false), 1)
	production := object(t, object(t, p["turn_report"])["production"])
	buildings := array(t, production["buildings_built"])
	equal(t, len(buildings), 1)
	built := checkShape(t, buildings[0], "building_id:n space:o zone:o")
	checkShape(t, built["space"], coordinateShape)
	checkShape(t, built["zone"], coordinateShape)
	equal(t, built["building_id"], float64(202))
	equal(t, built["zone"], map[string]any{"x": float64(0), "y": float64(1)})
	equal(t, production["techs_researched"], []any{float64(301)})
	equal(t, p["researched_techs"], map[string]any{"301": true, "302": true})
}

func checkShape(t *testing.T, value any, contract string) map[string]any {
	t.Helper()
	result := object(t, value)
	fields := strings.Fields(contract)
	if len(result) != len(fields) {
		t.Errorf("want keys %s, got %v", contract, reflect.ValueOf(result).MapKeys())
	}
	for _, field := range fields {
		key, kind, _ := strings.Cut(field, ":")
		item, exists := result[key]
		if !exists || !jsonKind(item, kind) {
			t.Errorf("field %s: want JSON kind %s, got %T (%v), present %t", key, kind, item, item, exists)
		}
	}
	return result
}

func jsonKind(value any, kind string) bool {
	switch kind {
	case "n":
		_, ok := value.(float64)
		return ok
	case "s":
		_, ok := value.(string)
		return ok
	case "b":
		_, ok := value.(bool)
		return ok
	case "o":
		_, ok := value.(map[string]any)
		return ok
	case "o?":
		return value == nil || jsonKind(value, "o")
	case "a":
		_, ok := value.([]any)
		return ok
	}
	return false
}

func equal(t *testing.T, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func snapshot(g *harness.Game, human int, viewer bool) map[string]any {
	g.T.Helper()
	data, err := json.Marshal(g.Risq.ToFrontend(uint64(human+1), viewer))
	if err != nil {
		g.T.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		g.T.Fatal(err)
	}
	return result
}

func object(t *testing.T, value any) map[string]any {
	t.Helper()
	result, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("want JSON object, got %T (%v)", value, value)
	}
	return result
}

func array(t *testing.T, value any) []any {
	t.Helper()
	result, ok := value.([]any)
	if !ok {
		t.Fatalf("want JSON array, got %T (%v)", value, value)
	}
	return result
}
