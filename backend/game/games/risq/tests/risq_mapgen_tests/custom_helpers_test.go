package risq_mapgen_tests

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq"
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
