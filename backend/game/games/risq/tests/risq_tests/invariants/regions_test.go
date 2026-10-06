package invariants

import "testing"

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
