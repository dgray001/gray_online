package invariants

import "testing"

func compareTrajectory(t *testing.T, want, got []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("trajectory length %d, want %d", len(got), len(want))
	}
	for turn := range want {
		if got[turn] == want[turn] {
			continue
		}
		i := 0
		for i < min(len(got[turn]), len(want[turn])) && got[turn][i] == want[turn][i] {
			i++
		}
		start := max(0, i-80)
		t.Fatalf("state %d differs at byte %d:\n got ...%s\nwant ...%s", turn, i, got[turn][start:min(len(got[turn]), i+160)], want[turn][start:min(len(want[turn]), i+160)])
	}
}
