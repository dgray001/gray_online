package risq_mapgen_tests

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

func TestCustomMapKeepsOnlyListedSpaces(t *testing.T) {
	board := mustGenerate(t, customDoc(1, "", `{"x":0,"y":0},{"x":1,"y":0,"terrain":6}`), 1)
	spaces := board.Inspect()
	if len(spaces) != 2 {
		t.Fatalf("%d spaces, want 2", len(spaces))
	}
	byCoord := map[[2]int]uint32{}
	for _, s := range spaces {
		byCoord[[2]int{s.Coord.X, s.Coord.Y}] = s.Terrain
		if len(s.Zones) != 7 {
			t.Errorf("space %v has %d zones, want 7", s.Coord, len(s.Zones))
		}
	}
	if byCoord[[2]int{0, 0}] != defs.DefaultTerrainId || byCoord[[2]int{1, 0}] != 6 {
		t.Errorf("terrains = %v, want default and 6", byCoord)
	}
}

func TestCustomMapRejectsBadSpaces(t *testing.T) {
	mustFail(t, customDoc(1, "", `{"x":0,"y":0},{"x":0,"y":0}`), 1, "off the board or listed twice")
	mustFail(t, customDoc(1, "", `{"x":5,"y":5}`), 1, "off the board or listed twice")
	mustFail(t, customDoc(1, "", `{"x":0,"y":0,"terrain":9999}`), 1, "unknown terrain")
}

func TestCustomMapRejectsBadHeader(t *testing.T) {
	mustFail(t, customDoc(0, "", hex7), 1, "players must be 1 to 12")
	mustFail(t, customDoc(13, "", hex7), 1, "players must be 1 to 12")
	mustFail(t, customDoc(2, "", hex7), 3, "2 player slots, got 3 players")
	mustFail(t, customDoc(1, `,"bogus":1`, hex7), 1, "unknown field")
	mustFail(t, `{"board_size":1,"players":1,"spaces":[{"x":0,"y":0,"zonez":[]}]}`, 1, "unknown field")
}
