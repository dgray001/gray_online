package risq_mapgen_tests

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/internal/mapgen"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

// Runs a custom map document as a launched game does: the given setup settings and a seeded rng
func seededBoard(t *testing.T, doc string, players int, seed int64, settings map[string]any) *fakeboard.Board {
	t.Helper()
	board := fakeboard.New()
	err := mapgen.GenerateCustomDocumentSeeded(board, rand.New(rand.NewSource(seed)), mapgen.ResolveSettings(settings, players), players, []byte(doc))
	if err != nil {
		t.Fatalf("%v\n%s", err, doc)
	}
	return board
}

func TestCustomMapExpressionsUseSetupSettings(t *testing.T) {
	extra := `,"starting_bank":{"gold":"map_resources * 10"},"space_gold_income":"starting_development"`
	for name, tc := range map[string]struct {
		settings    map[string]any
		gold, daily float64
	}{
		"defaults": {nil, 20, 2},
		"rich":     {map[string]any{"map_resources": float64(3), "starting_development": float64(4)}, 30, 4},
	} {
		t.Run(name, func(t *testing.T) {
			board := seededBoard(t, customDoc(2, extra, hex7), 2, 1, tc.settings)
			if board.Banks[0].Gold != tc.gold || board.Banks[1].Gold != tc.gold || goldIncome(t, board) != tc.daily {
				t.Errorf("banks %v, income %v; want gold %v and income %v", board.Banks, board.GoldIncome, tc.gold, tc.daily)
			}
		})
	}
}

func TestCustomMapUnitCountsUseSetupSettings(t *testing.T) {
	space := `{"x":0,"y":0,"zones":[{"x":0,"y":0,"units":[{"id":1,"player":0,"count":"starting_development * 2"}]}]}`
	board := seededBoard(t, customDoc(1, "", space), 1, 1, map[string]any{"starting_development": float64(3)})
	if units := zoneAt(t, board, 0, 0, 0, 0).Units; len(units) != 6 {
		t.Errorf("%d units, want starting_development 3 x 2", len(units))
	}
}

func TestCustomMapRejectsBadExpressions(t *testing.T) {
	mustFail(t, customDoc(2, `,"starting_bank":{"gold":"nonsense_variable"}`, hex7), 2, "unknown variable")
	mustFail(t, customDoc(2, `,"space_gold_income":"1 +"`, hex7), 2, "unexpected")
}

func TestCustomMapAppliesRulesToPlayingSlotsOnly(t *testing.T) {
	extra := `,"starting_bank":{"food":5,"wood":6,"stone":7,"gold":8},"starting_techs":[4],"unlimited_population":true,` +
		`"space_gold_income":30,"mercenaries_need_region":false`
	board := mustGenerate(t, customDoc(4, extra, hex7), 2)
	if len(board.Banks) != 2 || board.Banks[0].Gold != 8 || board.Banks[1].Food != 5 {
		t.Errorf("banks = %v, want food 5 and gold 8 for players 0 and 1 only", board.Banks)
	}
	if len(board.Techs) != 2 || board.Techs[1][0] != 4 {
		t.Errorf("techs = %v, want tech 4 for players 0 and 1 only", board.Techs)
	}
	if !board.Unlimited || goldIncome(t, board) != 30 || board.NeedRegion == nil || *board.NeedRegion {
		t.Errorf("unlimited %t, need region %v; want true and false", board.Unlimited, board.NeedRegion)
	}
}

func TestCustomMapRulesDefaultWhenOmitted(t *testing.T) {
	board := mustGenerate(t, customDoc(2, "", hex7), 2)
	if len(board.Banks)+len(board.Techs) != 0 || board.Unlimited || board.GoldIncome != nil || board.NeedRegion != nil {
		t.Errorf("omitted rules changed defaults: %v %v %t %v %v", board.Banks, board.Techs, board.Unlimited, board.GoldIncome, board.NeedRegion)
	}
}

func TestCustomMapRejectsUnknownStartingTech(t *testing.T) {
	mustFail(t, customDoc(2, `,"starting_techs":[999]`, hex7), 2, "unknown starting tech")
}

func TestCustomSpaceNamesAreTrimmedAndLimited(t *testing.T) {
	doc := func(name string) string {
		return `{"board_size":1,"players":1,"spaces":[{"x":0,"y":0,"name":"` + name + `"},{"x":1,"y":0}]}`
	}
	board := mustGenerate(t, doc("  Brazil "), 1)
	named, plain := board.Space(game_utils.Coordinate2D{}), board.Space(game_utils.Coordinate2D{X: 1})
	if got := named.(interface{ Name() string }).Name(); got != "Brazil" {
		t.Errorf("name %q, want it trimmed to Brazil", got)
	}
	if got := plain.(interface{ Name() string }).Name(); got != "" {
		t.Errorf("unnamed space got name %q", got)
	}
	mustFail(t, doc(strings.Repeat("x", 41)), 1, "name is longer than 40")
}

func TestCustomMapKeepsRegionRuleWhenFlagIsTrue(t *testing.T) {
	board := mustGenerate(t, customDoc(1, `,"mercenaries_need_region":true`, hex7), 1)
	if board.NeedRegion == nil || !*board.NeedRegion {
		t.Errorf("need region = %v, want an explicit true", board.NeedRegion)
	}
}
