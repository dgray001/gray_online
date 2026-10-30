package unit

import (
	"testing"

	. "github.com/dgray001/gray_online/game/games/risq/ai"
)

func unitsOnRegionSpaces() selectTestView {
	view := regionView()
	for x := 1; x <= 4; x++ {
		view.units = append(view.units, villager(uint64(x), Coordinate{X: x}))
	}
	return view
}

func TestZonePathReadsTheItemsOwnZone(t *testing.T) {
	view := unitsOnRegionSpaces()
	view.zones = []ZoneInfo{
		{Location: ZoneRef{Space: Coordinate{X: 1}}, HasResource: true, BuildingPlayer: -1},
		{Location: ZoneRef{Space: Coordinate{X: 2}}, BuildingPlayer: 1},
	}
	config := rule(`{"action":"set_var","name":"zones","value":{"sum":{"kind":"units","of":"var(item.zone.has_resource) * 10 + var(item.zone.building_player)"}}}`)
	if got := decide(t, config, view, "zones")["zones"]; got != 8 {
		t.Errorf("zone path sum = %v, want 9 + 1 for the known zones and -1 each for the two unknown ones", got)
	}
}

func TestLocationPathsValidateAtLoad(t *testing.T) {
	for _, config := range []string{
		rule(`{"action":"set_var","name":"x","value":{"sum":{"kind":"spaces","of":"var(item.zone.has_resource)"}}}`),
		rule(`{"action":"set_var","name":"x","value":{"sum":{"kind":"units","of":"var(item.space.no_such_field)"}}}`),
		rule(`{"action":"for_each","source":"units","as":"u","do":[{"action":"set_var","name":"x","value":"var(u.space.owner)"}]}`),
	} {
		if err := loadError(config); err == nil {
			t.Errorf("invalid path accepted: %s", config)
		}
	}
	valid := rule(`{"action":"set_var","name":"x","value":{"sum":{"kind":"spaces","of":"var(item.region.owner)"}}},
		{"action":"select","pick":{"kind":"zones","limit":1,"rank":"var(target.space.build_speed_modifier) + var(target.region.gold_bonus)"},"into":{"set":"best"}}`)
	if err := loadError(valid); err != nil {
		t.Errorf("valid paths rejected: %v", err)
	}
}
