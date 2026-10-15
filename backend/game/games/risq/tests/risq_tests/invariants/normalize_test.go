package invariants

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func canonical(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return string(data)
}

type canonicalItem struct {
	value any
	key   string
}

func unordered(key string, data []any) bool {
	if key == "spaces" {
		return len(data) > 0 && isNumber(data[0])
	}
	return slices.Contains([]string{"players", "units", "buildings", "resources", "subjects", "garrisoned_units", "planned_foundations", "available_mercenaries", "regions", "scores", "units_created", "units_hired", "buildings_built", "techs_researched", "combat", "failures"}, key)
}

func isNumber(value any) bool { _, ok := value.(float64); return ok }

func normalize(key string, value any) any {
	switch data := value.(type) {
	case object:
		delete(data, "game_base")
		if key == "player" {
			delete(data, "client_id")
			delete(data, "nickname")
		}
		for field, item := range data {
			data[field] = normalize(field, item)
		}
	case []any:
		for i, item := range data {
			data[i] = normalize("", item)
		}
		if unordered(key, data) {
			items := make([]canonicalItem, len(data))
			for i, item := range data {
				items[i] = canonicalItem{value: item, key: canonical(item)}
			}
			slices.SortFunc(items, func(a, b canonicalItem) int { return strings.Compare(a.key, b.key) })
			for i, item := range items {
				data[i] = item.value
			}
		}
	}
	return value
}

type object = map[string]any

func jsonValue(t *testing.T, value any) any {
	t.Helper()
	return jsonCompatible(reflect.ValueOf(value))
}

func jsonCompatible(value reflect.Value) any {
	if !value.IsValid() {
		return nil
	}
	if value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}
		return jsonCompatible(value.Elem())
	}
	switch value.Kind() {
	case reflect.Map:
		if value.IsNil() {
			return nil
		}
		result := object{}
		for _, key := range value.MapKeys() {
			result[key.String()] = jsonCompatible(value.MapIndex(key))
		}
		return result
	case reflect.Slice, reflect.Array:
		if value.Kind() == reflect.Slice && value.IsNil() {
			return nil
		}
		result := make([]any, value.Len())
		for i := range result {
			result[i] = jsonCompatible(value.Index(i))
		}
		return result
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(value.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(value.Uint())
	case reflect.Float32, reflect.Float64:
		return value.Float()
	default:
		return value.Interface()
	}
}

func snapshots(g *harness.Game, players int) []object {
	g.T.Helper()
	views := make([]object, players)
	for slot := range views {
		views[slot] = jsonValue(g.T, g.Risq.ToFrontend(uint64(g.Human(slot)+1), false)).(object)
	}
	return views
}

func self(view object, slot int) object {
	for _, entry := range view["players"].([]any) {
		player := entry.(object)
		if player["player"].(object)["player_id"] == float64(slot) {
			return player
		}
	}
	panic("player missing from own snapshot")
}

func TestNormalizationIgnoresSetOrderAndLobbyIdentity(t *testing.T) {
	a := object{"player": object{"player_id": 0, "client_id": 1, "nickname": "first"}, "game_base": object{"time": 1}, "units": []any{object{"internal_id": 2}, object{"internal_id": 1}}, "regions": []any{object{"spaces": []any{2.0, 1.0}}}}
	b := object{"player": object{"player_id": 0, "client_id": 2, "nickname": "second"}, "game_base": object{"time": 2}, "units": []any{object{"internal_id": 1}, object{"internal_id": 2}}, "regions": []any{object{"spaces": []any{1.0, 2.0}}}}
	if canonical(normalize("", a)) != canonical(normalize("", b)) {
		t.Fatal("equivalent states normalized differently")
	}
	if len(a["units"].([]any)) != 2 || a["player"].(object)["player_id"] != 0 {
		t.Fatal("normalization discarded entities or slot identity")
	}
}

func TestNormalizationPreservesGameplayOrder(t *testing.T) {
	for _, key := range []string{"active_orders", "production_queue", "move_path", "target_priority", "spaces", "zones"} {
		t.Run(key, func(t *testing.T) {
			a := object{key: []any{object{"id": 1}, object{"id": 2}}}
			b := object{key: []any{object{"id": 2}, object{"id": 1}}}
			if canonical(normalize("", a)) == canonical(normalize("", b)) {
				t.Fatalf("normalization erased %s order", key)
			}
		})
	}
}

func TestNormalizationRetainsGameplayFields(t *testing.T) {
	for _, key := range []string{"internal_id", "health", "current_stamina", "ownership", "resources_left", "target_id", "player_id", "population_limit"} {
		a, b := object{key: 1}, object{key: 2}
		if canonical(normalize("", a)) == canonical(normalize("", b)) {
			t.Errorf("normalization discarded %s", key)
		}
	}
}
