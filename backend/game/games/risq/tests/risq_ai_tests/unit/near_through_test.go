package unit

import (
	"slices"
	"testing"

	. "github.com/dgray001/gray_online/game/games/risq/ai"
)

type pathTestView struct {
	selectTestView
	neighbors map[Coordinate][]Coordinate
}

func (v pathTestView) SpaceNeighbors(space Coordinate) []Coordinate {
	return slices.Clone(v.neighbors[space])
}

func (v pathTestView) SpaceDistance(a Coordinate, b Coordinate) int {
	return 1
}

func TestNearThroughSemantics(t *testing.T) {
	// Board layout:
	// 0 - 1(water) - 2
	// 0 has link to 2
	spaces := []SpaceInfo{
		{Space: Coordinate{X: 0}, Passable: true},
		{Space: Coordinate{X: 1}, Passable: false}, // water/impassable
		{Space: Coordinate{X: 2}, Passable: true},
	}
	neighbors := map[Coordinate][]Coordinate{
		{X: 0}: {{X: 1}},
		{X: 1}: {{X: 0}, {X: 2}},
		{X: 2}: {{X: 1}},
	}
	view := pathTestView{
		selectTestView: selectTestView{
			loopTestView: loopTestView{spaces: spaces},
		},
		neighbors: neighbors,
	}

	// 1. Blocked by water (distance 2, through passable)
	blockedRule := rule(`{"action":"select","pick":{"kind":"spaces","where":{"near":{"distance":2,"of":{"x":0,"y":0},"through":{"if":"var(place.passable)"}}}},"into":{"set":"reached","var":"n"}}`)
	res := decide(t, blockedRule, view, "n")
	if res["n"] != 1 { // only anchor 0 is reachable (distance 0)
		t.Errorf("want only anchor reached (1), got %v", res["n"])
	}

	// 2. Add seam link between 0 and 2
	neighborsWithLink := map[Coordinate][]Coordinate{
		{X: 0}: {{X: 1}, {X: 2}},
		{X: 1}: {{X: 0}, {X: 2}},
		{X: 2}: {{X: 0}, {X: 1}},
	}
	viewWithLink := pathTestView{
		selectTestView: selectTestView{
			loopTestView: loopTestView{spaces: spaces},
		},
		neighbors: neighborsWithLink,
	}
	resLink := decide(t, blockedRule, viewWithLink, "n")
	if resLink["n"] != 2 { // anchor 0 and space 2
		t.Errorf("want 2 spaces reached with seam link, got %v", resLink["n"])
	}

	// 3. Anchor exempt even when anchor itself fails through
	// Anchor at space 1 (which has passable == false)
	anchorOnWater := rule(`{"action":"select","pick":{"kind":"spaces","where":{"near":{"distance":2,"of":{"x":1,"y":0},"through":{"if":"var(place.passable)"}}}},"into":{"set":"reached","var":"n"}}`)
	resAnchor := decide(t, anchorOnWater, view, "n")
	if resAnchor["n"] != 3 { // anchor 1 is at d=0, neighbors 0 and 2 are passable
		t.Errorf("anchor exempt: want 3 spaces reached, got %v", resAnchor["n"])
	}

	// 4. Empty through result excludes everything except anchor
	emptyThrough := rule(`{"action":"select","pick":{"kind":"spaces","where":{"near":{"distance":2,"of":{"x":0,"y":0},"through":{"if":"0"}}}},"into":{"set":"reached","var":"n"}}`)
	resEmpty := decide(t, emptyThrough, view, "n")
	if resEmpty["n"] != 1 {
		t.Errorf("empty through: want 1 (only anchor), got %v", resEmpty["n"])
	}
}
