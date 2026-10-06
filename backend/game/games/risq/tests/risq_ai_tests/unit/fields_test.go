package unit

import (
	"fmt"
	. "github.com/dgray001/gray_online/game/games/risq/ai"
	"strings"
	"testing"
)

func TestLoopItemFields(t *testing.T) {
	count := 2
	view := loopTestView{players: []int{0},
		units:     []UnitView{{InternalID: 7, UnitID: 11, Type: UnitTypeInfantry, Kind: UnitMilitary, Health: 4, MaxHealth: 8, CurrentStamina: 3, Location: ZoneRef{Space: Coordinate{X: 1}}}},
		buildings: []BuildingView{{InternalID: 9, BuildingID: 1, UnderConstruction: true, Health: 30, MaxHealth: 50}},
		spaces:    []SpaceInfo{{Space: Coordinate{X: 1}, Vision: 3, UnitCount: &count}},
		resources: map[ResourceCategory][]ResourceView{ResourceFood: {{InternalID: 5, AmountLeft: 12}}},
		zones:     []ZoneInfo{{HasResource: true}},
	}
	place := "x y zone_x zone_y distance_home"
	for source, c := range map[string]struct {
		fields string
		want   float64
	}{
		"players":   {"id is_me score units_visible buildings_known", 3},
		"spaces":    {"vision owner unit_count unidentified_count " + place, 8},
		"units":     {"id player unit_id unit_type is_military is_garrisoned health max_health stamina " + place, 38},
		"buildings": {"id player building_id is_under_construction health max_health " + place, 91},
		"resources": {"id category amount_left " + place, 17},
		"zones":     {"has_resource building_player " + place, 1},
	} {
		t.Run(source, func(t *testing.T) {
			var terms []string
			for _, field := range strings.Fields(c.fields) {
				terms = append(terms, "var(item."+field+")")
			}
			config := rule(fmt.Sprintf(`{"action":"for_each","source":%q,"as":"item","do":[{"action":"set_var","name":"sum","value":%q}]}`, source, strings.Join(terms, " + ")))
			if got := decide(t, config, view, "sum")["sum"]; got != c.want {
				t.Fatalf("field sum=%v, want %v", got, c.want)
			}
			bad := rule(fmt.Sprintf(`{"action":"for_each","source":%q,"as":"item","do":[{"action":"set_var","name":"bad","value":"var(item.no_such_field)"}]}`, source))
			if loadError(bad) == nil {
				t.Fatal("unknown item field accepted")
			}
		})
	}
}
