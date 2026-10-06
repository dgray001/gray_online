package risq_mapgen_tests

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

func zoneAt(t *testing.T, board *fakeboard.Board, x, y, zx, zy int) fakeboard.ZoneInfo {
	t.Helper()
	for _, s := range board.Inspect() {
		if s.Coord.X == x && s.Coord.Y == y {
			for _, z := range s.Zones {
				if z.Local.X == zx && z.Local.Y == zy {
					return z
				}
			}
		}
	}
	t.Fatalf("no zone (%d,%d) in space (%d,%d)", zx, zy, x, y)
	return fakeboard.ZoneInfo{}
}

func TestCustomMapPlacesZoneContents(t *testing.T) {
	doc := customDoc(2, "", `{"x":0,"y":0,"zones":[
		{"x":0,"y":0,"terrain_override":6,"building":{"id":1,"player":1},"units":[{"id":1,"player":1,"count":3},{"id":11,"player":0,"count":1}]},
		{"x":1,"y":0,"resource":41}]}`)
	board := mustGenerate(t, doc, 2)
	center, edge := zoneAt(t, board, 0, 0, 0, 0), zoneAt(t, board, 0, 0, 1, 0)
	if center.Override != 6 || center.Building.ID != 1 || center.Building.Player != 1 || len(center.Units) != 4 {
		t.Errorf("center = %+v, want override 6, building 1 for player 1, 4 units", center)
	}
	if edge.Resource != 41 || edge.Building.ID != 0 {
		t.Errorf("edge = %+v, want resource 41 only", edge)
	}
}

func TestCustomMapSkipsPlayersBeyondTheGame(t *testing.T) {
	doc := customDoc(4, "", `{"x":0,"y":0,"zones":[
		{"x":0,"y":0,"building":{"id":1,"player":3},"units":[{"id":1,"player":2,"count":2},{"id":1,"player":1,"count":1}]}]}`)
	center := zoneAt(t, mustGenerate(t, doc, 2), 0, 0, 0, 0)
	if center.Building.ID != 0 || len(center.Units) != 1 || center.Units[0].Player != 1 {
		t.Errorf("center = %+v, want only player 1's unit", center)
	}
}

func TestCustomMapRejectsBadZoneContents(t *testing.T) {
	zone := func(body string) string { return customDoc(1, "", `{"x":0,"y":0,"zones":[{`+body+`}]}`) }
	mustFail(t, zone(`"x":3,"y":3`), 1, "has no zone")
	mustFail(t, zone(`"x":0,"y":0,"resource":9999`), 1, "unknown or its zone is occupied")
	mustFail(t, zone(`"x":0,"y":0,"building":{"id":9999,"player":0}`), 1, "could not place building")
	mustFail(t, zone(`"x":0,"y":0,"resource":41,"building":{"id":1,"player":0}`), 1, "could not place building")
	mustFail(t, zone(`"x":0,"y":0,"units":[{"id":9999,"player":0,"count":1}]`), 1, "unknown or has a negative player")
	mustFail(t, zone(`"x":0,"y":0,"units":[{"id":1,"player":-1,"count":1}]`), 1, "unknown or has a negative player")
}
