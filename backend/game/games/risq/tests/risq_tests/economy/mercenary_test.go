package economy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

const mercenaryRules = `,"starting_techs":[4],"space_gold_income":0`

const configDirMapRules = `"board_size":2,"players":2,"space_gold_income":0,"starting_bank":{"gold":500}`

const configDirSpaces = `"spaces":[{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":11,"player":0,"count":1}]}]},{"x":-2,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":11,"player":1,"count":1}]}]}]`

var mercenaryOnlyConfig = map[string]string{
	"buildings.json": `[]`,
	"units.json":     `[{"unit_id":11,"display_name":"Unit","unit_type":"infantry","max_health":11,"attack_type":"blunt","attack_blunt":6,"cost":{"gold":2},"turn_stamina":15}]`,
	"techs.json":     `[{"tech_id":4,"display_name":"Mercenary Contracts","research_stamina":45,"unlocks_mercenary_ids":[11,12]}]`,
	"bonuses.json":   `[{"tech_id":1,"mercenary_cost_markup_delta":-0.3}]`,
}

func TestMapConfigDirScopesTablesMercenariesAndPrice(t *testing.T) {
	risk := `{"config_dir":"mercenary_only","starting_techs":[4],"unlimited_population":true,` + configDirMapRules + `,` + configDirSpaces + `}`
	plain := `{"starting_techs":[4],` + configDirMapRules + `,` + configDirSpaces + `}`
	dir := fakeboard.UseConfig(t, testConfig, nil, map[string]string{"risk": risk, "plain": plain})
	if err := os.MkdirAll(filepath.Join(dir, "mercenary_only"), 0o755); err != nil {
		t.Fatal(err)
	}
	for file, data := range mercenaryOnlyConfig {
		if err := os.WriteFile(filepath.Join(dir, "mercenary_only", file), []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	g := startGame(t, "custom:risk", 1, 2)
	other := startGame(t, "custom:plain", 1, 2)
	p := g.Human(0)
	if hireable := g.Self(p).AvailableMercenaries; len(hireable) != 1 || hireable[0].ID != 11 {
		t.Fatalf("mercenaries %v; want only unit 11 since unit 12 is missing from the config", hireable)
	}
	if len(g.Risq.TechTreeToFrontend()) != 0 || len(other.Risq.TechTreeToFrontend()) == 0 {
		t.Error("tech tree did not follow each game's own config")
	}
	if len(other.Self(other.Human(0)).AvailableMercenaries) != 2 {
		t.Error("a map without config_dir lost the default mercenaries")
	}
	g.Submit(p, harness.Order(defs.OrderType_BuyMercenary, nil, harness.BuildKey(11, 0, 0, 0, 0), false))
	g.EndTurn()
	state := g.Self(p)
	if len(state.Units) != 2 || state.Resources.Gold != 497 {
		t.Errorf("hiring: units %d, gold %v; want two units and a 3 gold price (2 gold unit x 1.5 gold markup x 1.0)", len(state.Units), state.Resources.Gold)
	}
}

func TestMercenaryHiringChargesGoldAndPlacesUnit(t *testing.T) {
	g := productionGame(t, 1, 1, mercenaryRules)
	p := g.Human(0)
	g.Submit(p, harness.Order(defs.OrderType_BuyMercenary, nil, harness.BuildKey(11, 0, 0, 0, 0), false))
	g.EndTurn()
	state := g.Self(p)
	if len(state.Units) != 2 || state.Resources.Gold != 409 || state.Resources.Food != 300 || state.Resources.Wood != 300 {
		t.Fatalf("hiring: units %d, resources %+v; want two units and gold-only charge 91", len(state.Units), state.Resources)
	}
	hired := 0
	for _, u := range state.Units {
		if u.UnitID == 11 && u.Space == (harness.Coord{}) && u.Zone == (harness.Coord{}) {
			hired++
		}
	}
	if hired != 1 {
		t.Errorf("hired %d infantry in requested zone, want one", hired)
	}
	g.EndTurn()
	if state = g.Self(p); len(state.Units) != 2 || state.Resources.Gold != 409 {
		t.Error("hiring repeated on the following turn")
	}
}
