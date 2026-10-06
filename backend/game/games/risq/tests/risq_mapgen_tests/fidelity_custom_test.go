package risq_mapgen_tests

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

// Every custom map document must build the same board, or fail the same way, in the fake and the engine
func checkCustomDocMatches(t *testing.T, doc string) {
	t.Helper()
	var header struct{ Players int }
	json.Unmarshal([]byte(doc), &header)
	board, fakeErr := fakeboard.GenerateDocument(doc, header.Players)
	checkCustomBoardMatches(t, doc, board, fakeErr)
}

func checkCustomBoardMatches(t *testing.T, doc string, board *fakeboard.Board, fakeErr error) {
	t.Helper()
	engine, engineErr := enginePreview(t, doc)
	if fmt.Sprint(fakeErr) != fmt.Sprint(engineErr) {
		t.Fatalf("fake error %v, engine error %v", fakeErr, engineErr)
	}
	if fakeErr != nil {
		return
	}
	if !reflect.DeepEqual(fakeViews(board), engine.Spaces) {
		t.Error("fake and engine spaces differ")
	}
	if !slices.Equal(board.LinkLines(), engine.Links) {
		t.Errorf("links differ: fake %v, engine %v", board.LinkLines(), engine.Links)
	}
	if !slices.Equal(board.RegionLines(), engine.Regions) {
		t.Errorf("regions differ: fake %v, engine %v", board.RegionLines(), engine.Regions)
	}
	if board.GoldIncome != nil && !reflect.DeepEqual(engine.Incomes, map[float64]bool{*board.GoldIncome: true}) {
		t.Errorf("gold income: fake %v, engine %v", *board.GoldIncome, engine.Incomes)
	}
}

// Large maps cost the engine over a second to preview, so only small ones are compared; the tiny docs below cover the features
func TestShippedCustomMapsGenerate(t *testing.T) {
	entries, err := defs.ListConfigDir("maps", "custom")
	if err != nil || len(entries) == 0 {
		t.Fatalf("no shipped custom maps: %v", err)
	}
	for _, entry := range entries {
		name := strings.TrimSuffix(entry.Name(), ".json")
		t.Run(name, func(t *testing.T) {
			doc, err := defs.ReadConfigFile("maps", "custom", entry.Name())
			if err != nil {
				t.Fatal(err)
			}
			var header struct {
				Players int
				Spaces  []json.RawMessage
			}
			if err := json.Unmarshal(doc, &header); err != nil {
				t.Fatal(err)
			}
			board := mustGenerate(t, string(doc), header.Players)
			if len(board.Violations) > 0 {
				t.Errorf("placement violations: %v", board.Violations)
			}
			if len(header.Spaces) <= 100 {
				checkCustomBoardMatches(t, string(doc), board, nil)
			}
		})
	}
}

func TestFakeBoardMatchesEngineOnTinyDocs(t *testing.T) {
	two := `{"x":-3,"y":0},{"x":3,"y":0}`
	docs := map[string]string{
		"starts":        `{"board_size":3,"players":2,"starting_bank":{"food":1,"wood":2,"stone":3,"gold":4},` + startTemplate + `,"spaces":[` + hexSpaces(3, map[[2]int]int{{-2, 0}: 0, {1, 0}: 1}) + `]}`,
		"footprint":     `{"board_size":2,"players":2,` + startTemplate + `,"spaces":[{"x":-2,"y":0,"player_slot":0,"zones":[{"x":1,"y":0,"resource":41},{"x":0,"y":0,"units":[{"id":1,"player":1,"count":2}]}]},{"x":2,"y":0,"player_slot":1}]}`,
		"seam":          `{"board_size":3,"players":1,"spaces":[` + two + `],"connections":[{"from":[-3,0],"to":[3,0],"direction":3}]}`,
		"center link":   `{"board_size":3,"players":1,"spaces":[` + two + `],"connections":[{"from":[-3,0],"to":[3,0]}]}`,
		"regions+gold":  `{"board_size":3,"players":1,"space_gold_income":30,"regions":[{"name":"A","gold_bonus":5,"spaces":[[-3,0],[3,0]]}],"spaces":[` + two + `]}`,
		"zone contents": customDoc(2, "", `{"x":0,"y":0,"zones":[{"x":0,"y":0,"terrain_override":6,"building":{"id":1,"player":1},"units":[{"id":1,"player":0,"count":2}]},{"x":1,"y":0,"resource":41}]}`),
	}
	for name, doc := range docs {
		t.Run(name, func(t *testing.T) { checkCustomDocMatches(t, doc) })
	}
}
