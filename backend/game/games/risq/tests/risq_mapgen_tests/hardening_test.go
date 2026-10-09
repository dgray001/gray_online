package risq_mapgen_tests

import (
	"strings"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

func TestGenerateReturnsAnErrorForNonPositivePlayerCounts(t *testing.T) {
	for _, players := range []int{0, -1} {
		for _, name := range []string{"custom:tiny", "script:ring", "script:triangle"} {
			if _, err := fakeboard.Generate(name, players, 1); err == nil || !strings.Contains(err.Error(), "need at least 1 player") {
				t.Errorf("%s with %d players: error %v, want a player count error", name, players, err)
			}
		}
		if _, err := fakeboard.GenerateDocument(customDoc(1, "", hex7), players); err == nil {
			t.Errorf("a document with %d players should be rejected", players)
		}
	}
}

func TestScriptStepsLogUnknownParamsAndCarryOn(t *testing.T) {
	useScripts(t, map[string]string{"typo": `[` + step("shape", `{"kind":"hexagon","size":{"radius":3},"sise":9}`) + `,` + step("terrain_fill", `{"terrain_id":51,"colour":"red"}`) + `]`})
	board := generateStartless(t, "typo", 2)
	if len(board.Inspect()) != 37 || len(withTerrain(board, 51)) != 37 {
		t.Errorf("%d spaces, %d hilly; want the valid params applied to all 37", len(board.Inspect()), len(withTerrain(board, 51)))
	}
}

func TestRegionsSevenNeverReusesASuppliedName(t *testing.T) {
	useScripts(t, map[string]string{"named": `[` + step("shape", `{"kind":"hexagon","size":{"radius":4}}`) + `,` + step("regions_seven", `{"names":["Ashenvale","Blackmoor","Cindergate"]}`) + `]`})
	for seed := int64(1); seed <= 40; seed++ {
		board, err := fakeboard.Generate("script:named", 2, seed)
		if err == nil || err.Error() != startsMissing {
			t.Fatalf("seed %d: error %v, want only the missing starts error", seed, err)
		}
		names := map[string]bool{}
		for _, region := range board.Regions {
			names[region.Name] = true
		}
		if len(board.Regions) != 7 || len(names) != 7 {
			t.Fatalf("seed %d: %d regions with %d distinct names, want 7 and 7", seed, len(board.Regions), len(names))
		}
	}
}

func TestCustomMapIgnoresAnUnknownTerrainOverride(t *testing.T) {
	doc := customDoc(1, "", `{"x":0,"y":0,"zones":[{"x":0,"y":0,"terrain_override":9999},{"x":1,"y":0,"terrain_override":6}]}`)
	board := mustGenerate(t, doc, 1)
	if got := zoneAt(t, board, 0, 0, 0, 0).Override; got != 0 {
		t.Errorf("unknown override applied as %d, want it ignored", got)
	}
	if got := zoneAt(t, board, 0, 0, 1, 0).Override; got != 6 {
		t.Errorf("known override = %d, want 6", got)
	}
	checkCustomDocMatches(t, doc)
}

func TestCustomMapSkipsUnitsWithANegativeCount(t *testing.T) {
	doc := customDoc(1, "", `{"x":0,"y":0,"zones":[{"x":0,"y":0,"units":[{"id":1,"player":0,"count":-3},{"id":11,"player":0,"count":2}]}]}`)
	board := mustGenerate(t, doc, 1)
	if units := zoneAt(t, board, 0, 0, 0, 0).Units; len(units) != 2 || units[0].ID != 11 {
		t.Errorf("units = %v, want only the 2 of unit 11", units)
	}
	checkCustomDocMatches(t, doc)
}
