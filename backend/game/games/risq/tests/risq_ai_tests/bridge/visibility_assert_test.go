package bridge

import "testing"

func assertVisibility(t *testing.T, seen observation, x int, vision uint8) {
	t.Helper()
	found := false
	for _, space := range seen.Spaces {
		if space.Space.X == x && space.Space.Y == 0 {
			found = true
			if space.Vision != vision {
				t.Fatalf("vision=%d, want %d", space.Vision, vision)
			}
			if vision == 2 && (space.UnitCount == nil || *space.UnitCount != 1) {
				t.Fatalf("count=%+v", space)
			}
		}
	}
	if !found {
		t.Fatal("target space missing")
	}
	if vision == 0 {
		if len(seen.Units)+len(seen.VisibleBuildings)+len(seen.KnownBuildings)+len(seen.Resources) != 0 {
			t.Fatalf("unexplored leaked: %+v", seen)
		}
		return
	}
	if len(seen.Resources) != 1 || len(seen.VisibleBuildings) != 1 || len(seen.KnownBuildings) != 1 {
		t.Fatalf("known contents=%+v", seen)
	}
	if vision == 2 {
		if len(seen.Units) != 0 {
			t.Fatalf("count vision exposed units: %+v", seen.Units)
		}
		return
	}
	if len(seen.Units) != 1 {
		t.Fatalf("visible units=%+v", seen.Units)
	}
	if (seen.Units[0].CurrentOrder != nil) != (vision == 4) {
		t.Fatalf("enemy order at vision %d: %+v", vision, seen.Units[0])
	}
}
