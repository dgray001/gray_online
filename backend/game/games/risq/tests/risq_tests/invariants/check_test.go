package invariants

import (
	"fmt"
	"math"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func checkState(t *testing.T, g *harness.Game, players int, capped bool) []object {
	t.Helper()
	views := snapshots(g, players)
	items := inventoryOf(t, views)
	for slot, view := range views {
		checkPlayer(t, self(view, slot), slot, capped)
		checkBoard(t, view, items)
		checkRegions(t, view)
	}
	return views
}

func bound(t *testing.T, label string, value, low, high float64) {
	t.Helper()
	if math.IsNaN(value) || math.IsInf(value, 0) || value < low || value > high {
		t.Errorf("%s = %v, want [%v, %v]", label, value, low, high)
	}
}

func number(data object, key string) float64 { return data[key].(float64) }

func objects(data object, key string) []object {
	var result []object
	for _, value := range data[key].([]any) {
		result = append(result, value.(object))
	}
	return result
}

type inventory map[string]map[float64]object

func inventoryOf(t *testing.T, views []object) inventory {
	t.Helper()
	items := inventory{"units": {}, "buildings": {}}
	for slot, view := range views {
		for kind, entries := range items {
			for _, entity := range objects(self(view, slot), kind) {
				id := number(entity, "internal_id")
				if entries[id] != nil {
					t.Fatalf("duplicate global %s id %v", kind, id)
				}
				entries[id] = entity
			}
		}
	}
	return items
}

func checkAccounting(t *testing.T, player object) {
	t.Helper()
	report, ok := player["turn_report"].(object)
	if !ok {
		return
	}
	bank := player["resources"].(object)
	seen := map[int]bool{}
	for _, flow := range objects(report, "resources") {
		category := int(number(flow, "category"))
		if category < 1 || category > 4 || seen[category] {
			t.Fatalf("invalid or duplicate resource category %d", category)
		}
		seen[category] = true
		key := []string{"food", "wood", "stone", "gold"}[category-1]
		want := number(flow, "start") + number(flow, "gathered") + number(flow, "income") - number(flow, "spent")
		if math.Abs(number(bank, key)-want) > 0.000001 || math.Abs(number(flow, "final")-want) > 0.000001 {
			t.Errorf("%s bank %v disagrees with report %+v", key, bank[key], flow)
		}
	}
	if len(seen) != 4 {
		t.Errorf("resource report contains %d categories, want 4", len(seen))
	}
}

func checkPlayer(t *testing.T, player object, slot int, capped bool) {
	t.Helper()
	for key, value := range player["resources"].(object) {
		bound(t, key, value.(float64), 0, math.MaxFloat64)
	}
	checkAccounting(t, player)
	support := 0.0
	for _, building := range objects(player, "buildings") {
		if !building["under_construction"].(bool) {
			support += number(building, "population_support")
		}
	}
	limit := number(player, "population_limit")
	if limit != min(support, number(player, "max_population_limit")) {
		t.Errorf("player %d cap %v disagrees with support %v", slot, limit, support)
	}
	if capped && float64(len(objects(player, "units"))) > limit {
		t.Errorf("player %d exceeded population cap %v", slot, limit)
	}
	for _, kind := range []string{"units", "buildings"} {
		checkPlayerEntities(t, player, slot, kind)
	}
}

func checkPlayerEntities(t *testing.T, player object, slot int, kind string) {
	t.Helper()
	seen := map[float64]bool{}
	for _, entity := range objects(player, kind) {
		id := number(entity, "internal_id")
		if seen[id] || number(entity, "player_id") != float64(slot) {
			t.Errorf("%s %v duplicated or assigned to wrong player %d", kind, id, slot)
		}
		seen[id] = true
		checkEntityBounds(t, entity)
	}
}

func checkEntityBounds(t *testing.T, entity object) {
	t.Helper()
	label := fmt.Sprintf("entity %v", entity["internal_id"])
	stats := entity["combat_stats"].(object)
	bound(t, label+" health", number(stats, "health"), 0, number(stats, "max_health"))
	if number(stats, "health") == 0 {
		t.Errorf("%s survived with zero health", label)
	}
	bound(t, label+" stamina", number(entity, "current_stamina"), 0, number(entity, "max_stamina"))
	if _, ok := entity["building_id"]; !ok {
		return
	}
	bound(t, label+" construction", number(entity, "stamina_remaining"), 0, number(entity, "construction_stamina_total"))
	if _, ok := entity["resources_left"]; ok {
		bound(t, label+" resources", number(entity, "resources_left"), 0, number(entity, "resource_capacity"))
	}
}

func checkBoard(t *testing.T, view object, items inventory) {
	t.Helper()
	locations := map[string]map[float64]int{"units": {}, "buildings": {}, "resources": {}}
	for _, row := range view["spaces"].([]any) {
		for _, value := range row.([]any) {
			if value == nil {
				continue
			}
			space := value.(object)
			if number(space, "visibility") != 4 {
				t.Fatal("fixture must expose the entire board")
			}
			checkSpace(t, space, items, locations)
		}
	}
	checkGarrisons(t, items, locations["units"])
	for kind, entries := range items {
		for id := range entries {
			if locations[kind][id] != 1 {
				t.Errorf("%s %v has %d locations", kind, id, locations[kind][id])
			}
		}
	}
}

func checkRegions(t *testing.T, view object) {
	t.Helper()
	owners := map[float64]float64{}
	for _, row := range view["spaces"].([]any) {
		for _, value := range row.([]any) {
			if value != nil {
				s := value.(object)
				owners[number(s, "coordinate_key")] = number(s, "ownership")
			}
		}
	}
	for _, region := range objects(view, "regions") {
		candidates := map[float64]bool{}
		for _, key := range region["spaces"].([]any) {
			owner, ok := owners[key.(float64)]
			if !ok {
				t.Fatalf("region references absent space %v", key)
			}
			candidates[owner] = true
		}
		want := -1.0
		if len(candidates) == 1 {
			for owner := range candidates {
				want = owner
			}
		}
		if region["owner"] != want {
			t.Errorf("region %v owner %v, want %v", region["name"], region["owner"], want)
		}
	}
}

func checkSpace(t *testing.T, space object, items inventory, locations map[string]map[float64]int) {
	t.Helper()
	contents := map[string][]any{"units": {}, "buildings": {}, "resources": {}}
	for _, row := range space["zones"].([]any) {
		for _, value := range row.([]any) {
			if value == nil {
				continue
			}
			zone := value.(object)
			if zone["ownership"] != space["ownership"] {
				t.Error("zone ownership differs from space")
			}
			for _, unit := range objects(zone, "units") {
				checkZoneEntity(t, unit, space, zone, "units", items, locations)
				contents["units"] = append(contents["units"], unit)
			}
			if building, ok := zone["building"].(object); ok {
				checkZoneEntity(t, building, space, zone, "buildings", items, locations)
				contents["buildings"] = append(contents["buildings"], building)
			}
			if resource, ok := zone["resource"].(object); ok {
				id := number(resource, "internal_id")
				locations["resources"][id]++
				if locations["resources"][id] != 1 {
					t.Errorf("resource %v occurs in multiple zones", id)
				}
				checkResource(t, resource, space, zone)
				contents["resources"] = append(contents["resources"], resource)
			}
		}
	}
	for kind, entries := range contents {
		if canonical(normalize(kind, jsonValue(t, entries))) != canonical(normalize(kind, jsonValue(t, space[kind]))) {
			t.Errorf("space %v %s summary differs from zones", space["coordinate"], kind)
		}
	}
	checkOwnership(t, space)
}

func checkGarrisons(t *testing.T, items inventory, locations map[float64]int) {
	t.Helper()
	for id, building := range items["buildings"] {
		garrison := building["garrisoned_units"].([]any)
		bound(t, "garrison size", float64(len(garrison)), 0, number(building, "garrison_capacity"))
		if building["has_garrisoned_units"] != (len(garrison) > 0) {
			t.Errorf("building %v garrison flag disagrees", id)
		}
		for _, entry := range garrison {
			unitID := entry.(float64)
			locations[unitID]++
			unit := items["units"][unitID]
			if unit == nil {
				t.Fatalf("garrison %v contains orphan unit %v", id, unitID)
			}
			if unit["garrisoned_in"] != id || unit["player_id"] != building["player_id"] {
				t.Errorf("unit %v has inconsistent garrison %v", unitID, id)
			}
			if unit["space_coordinate"] != nil || unit["zone_coordinate"] != nil {
				t.Errorf("unit %v has garrison and zone coordinates", unitID)
			}
		}
	}
}

func expectedOwner(space object) float64 {
	owners := map[float64]bool{}
	for _, building := range objects(space, "buildings") {
		owners[number(building, "player_id")] = true
	}
	if len(owners) == 0 {
		for _, unit := range objects(space, "units") {
			if number(unit, "unit_type") != 1 {
				owners[number(unit, "player_id")] = true
			}
		}
	}
	if len(owners) == 1 {
		for owner := range owners {
			return owner
		}
	}
	return -1
}

func checkOwnership(t *testing.T, space object) {
	t.Helper()
	if want := expectedOwner(space); space["ownership"] != want {
		t.Errorf("space %v owner %v, want %v", space["coordinate"], space["ownership"], want)
	}
}

func checkResource(t *testing.T, resource, space, zone object) {
	t.Helper()
	config := defs.ResourceConfigs[uint32(number(resource, "resource_id"))]
	bound(t, "node resources", number(resource, "resources_left"), 0, config.Starting_resources)
	if number(resource, "resources_left") == 0 {
		t.Error("depleted resource node was not removed")
	}
	if canonical(resource["space_coordinate"]) != canonical(space["coordinate"]) || canonical(resource["zone_coordinate"]) != canonical(zone["coordinate"]) {
		t.Error("resource coordinates disagree with zone")
	}
}

func checkZoneEntity(t *testing.T, entity, space, zone object, kind string, items inventory, locations map[string]map[float64]int) {
	t.Helper()
	id := number(entity, "internal_id")
	locations[kind][id]++
	owned := items[kind][id]
	if owned == nil {
		t.Fatalf("zone contains orphan %s %v", kind, id)
	}
	for _, copy := range []object{entity, owned} {
		if canonical(copy["space_coordinate"]) != canonical(space["coordinate"]) || canonical(copy["zone_coordinate"]) != canonical(zone["coordinate"]) {
			t.Errorf("%s %v coordinates disagree with containing zone", kind, id)
		}
		if copy["garrisoned_in"] != nil {
			t.Errorf("garrisoned unit %v also occurs in a zone", id)
		}
	}
	if entity["player_id"] != owned["player_id"] {
		t.Errorf("%s %v zone owner differs from inventory", kind, id)
	}
	if kind == "buildings" && canonical(normalize("garrisoned_units", jsonValue(t, entity["garrisoned_units"]))) != canonical(normalize("garrisoned_units", jsonValue(t, owned["garrisoned_units"]))) {
		t.Errorf("building %v garrison differs between zone and owner", id)
	}
}
