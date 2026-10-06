package risq_mapgen_tests

import "testing"

const hexagon2 = `{"step":"shape","params":{"kind":"hexagon","size":2}}`

func TestRulesStepAppliesToEveryPlayer(t *testing.T) {
	useScripts(t, map[string]string{"rules": `[` + hexagon2 + `,
		{"step":"define","params":{"name":"x","value":"board_size * 3"}},
		{"step":"rules","params":{"space_gold_income":"x + 1","starting_techs":[4,1],"unlimited_population":true,"mercenaries_need_region":false}}]`})
	board := generateStartless(t, "rules", 3)
	if board.GoldIncome == nil || *board.GoldIncome != 7 {
		t.Errorf("gold income = %v, want 7 (defined x = 6, plus 1)", board.GoldIncome)
	}
	if !board.Unlimited || board.NeedRegion == nil || *board.NeedRegion {
		t.Errorf("unlimited = %t, need region = %v; want true and false", board.Unlimited, board.NeedRegion)
	}
	for player := 0; player < 3; player++ {
		if got := board.Techs[player]; len(got) != 2 || got[0] != 4 || got[1] != 1 {
			t.Errorf("player %d techs = %v, want [4 1]", player, got)
		}
	}
}

func TestRulesStepLeavesDefaultsWhenOmitted(t *testing.T) {
	useScripts(t, map[string]string{"plain": `[` + hexagon2 + `,{"step":"rules","params":{}}]`})
	board := generateStartless(t, "plain", 2)
	if board.GoldIncome != nil || board.Unlimited || board.NeedRegion != nil || len(board.Techs) != 0 {
		t.Errorf("empty rules changed defaults: %+v", board)
	}
}

func TestDefineChainsVariables(t *testing.T) {
	useScripts(t, map[string]string{"chain": `[` + hexagon2 + `,
		{"step":"define","params":{"name":"a","value":"max(num_players, 2) * 10"}},
		{"step":"define","params":{"name":"b","value":"a - board_size - min(1, 5)"}},
		{"step":"rules","params":{"space_gold_income":"b"}}]`})
	if got := goldIncome(t, generateStartless(t, "chain", 4)); got != 37 {
		t.Errorf("b = %v, want 37 (40 - 2 - 1)", got)
	}
}
