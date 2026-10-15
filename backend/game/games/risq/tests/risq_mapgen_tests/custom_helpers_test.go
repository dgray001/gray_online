package risq_mapgen_tests

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

// A radius-1 custom map document with the given extra top-level fields and space entries
func customDoc(players int, extra string, spaces string) string {
	return `{"board_size":1,"players":` + strconv.Itoa(players) + extra + `,"spaces":[` + spaces + `]}`
}

// Space entries for every space within radius, with player_slot set on the coordinates in slots
func hexSpaces(radius int, slots map[[2]int]int) string {
	var entries []string
	for r := -radius; r <= radius; r++ {
		for q := max(-radius, -radius-r); q <= min(radius, radius-r); q++ {
			entry := `{"x":` + strconv.Itoa(q) + `,"y":` + strconv.Itoa(r)
			if slot, ok := slots[[2]int{q, r}]; ok {
				entry += `,"player_slot":` + strconv.Itoa(slot)
			}
			entries = append(entries, entry+`}`)
		}
	}
	return strings.Join(entries, ",")
}

const hex7 = `{"x":0,"y":0},{"x":1,"y":0},{"x":0,"y":1},{"x":-1,"y":1},{"x":-1,"y":0},{"x":0,"y":-1},{"x":1,"y":-1}`

func mustGenerate(t *testing.T, doc string, players int) *fakeboard.Board {
	t.Helper()
	board, err := fakeboard.GenerateDocument(doc, players)
	if err != nil {
		t.Fatalf("%v\n%s", err, doc)
	}
	return board
}

// The fake must reject the doc, and so must the engine whenever it builds the same player count the fake did
func mustFail(t *testing.T, doc string, players int, want string) {
	t.Helper()
	if _, err := fakeboard.GenerateDocument(doc, players); err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("fake error %v, want it to contain %q\n%s", err, want, doc)
	}
	var header struct{ Players int }
	json.Unmarshal([]byte(doc), &header)
	if header.Players != players {
		return
	}
	if _, err := risq.PreviewCustomMap([]byte(doc)); err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("engine error %v, want it to contain %q\n%s", err, want, doc)
	}
}

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

func TestCustomMapBuildsRegions(t *testing.T) {
	regions := `,"regions":[{"name":"West","gold_bonus":3,"spaces":[[0,0],[-1,0]]},{"name":"East","spaces":[[1,0]]}]`
	board := mustGenerate(t, customDoc(1, regions, hex7), 1)
	if len(board.Regions) != 2 {
		t.Fatalf("%d regions, want 2", len(board.Regions))
	}
	if west := board.Regions[0]; west.Name != "West" || west.GoldBonus != 3 || len(west.Keys) != 2 {
		t.Errorf("west = %+v, want 2 spaces and a bonus of 3", west)
	}
	if east := board.Regions[1]; east.GoldBonus != 0 || len(east.Keys) != 1 {
		t.Errorf("east = %+v, want 1 space and no bonus", east)
	}
}

func TestCustomMapRejectsBadRegions(t *testing.T) {
	regions := func(body string) string { return `,"regions":[` + body + `]` }
	mustFail(t, customDoc(1, regions(`{"name":"A","spaces":[[9,9]]}`), hex7), 1, "is not on the map")
	mustFail(t, customDoc(1, regions(`{"name":"A","spaces":[[0,0]]},{"name":"B","spaces":[[0,0]]}`), hex7), 1, "already belongs to region")
	mustFail(t, customDoc(1, regions(`{"name":"A","spaces":[[0,0]]},{"name":"A","spaces":[[1,0]]}`), hex7), 1, "already exists")
}
