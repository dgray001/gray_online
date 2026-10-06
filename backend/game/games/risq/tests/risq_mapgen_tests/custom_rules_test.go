package risq_mapgen_tests

import "testing"

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

func TestCustomMapKeepsRegionRuleWhenFlagIsTrue(t *testing.T) {
	board := mustGenerate(t, customDoc(1, `,"mercenaries_need_region":true`, hex7), 1)
	if board.NeedRegion == nil || !*board.NeedRegion {
		t.Errorf("need region = %v, want an explicit true", board.NeedRegion)
	}
}
