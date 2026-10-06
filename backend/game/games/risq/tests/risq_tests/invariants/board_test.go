package invariants

import "testing"

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
