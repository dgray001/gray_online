package shape

import "testing"

func targetZoneContract(t *testing.T, zone map[string]any) string {
	t.Helper()
	coordinate := object(t, zone["coordinate"])
	x, y := coordinate["x"].(float64), coordinate["y"].(float64)
	contract := zoneShape
	if y == 0 && (x == 0 || x == -1) {
		contract += " building:o"
	}
	if y == 0 && x == 1 {
		contract += " resource:o terrain_override_display_name:s"
		equal(t, zone["terrain_override"], float64(24))
	} else {
		equal(t, zone["terrain_override"], float64(0))
	}
	return contract
}
