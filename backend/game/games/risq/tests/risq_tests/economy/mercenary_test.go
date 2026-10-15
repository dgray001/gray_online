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

func TestMercenaryBatchReservesGoldAndPopulation(t *testing.T) {
	for name, c := range map[string]struct {
		extra, reason string
		hired         int
	}{"capacity": {"", "population capped", 4}, "affordability": {`,"unlimited_population":true`, "cannot afford mercenary", 5}} {
		t.Run(name, func(t *testing.T) {
			g := productionGame(t, 1, 1, mercenaryRules+c.extra)
			p := g.Human(0)
			orders := make([]defs.OrderFromFrontend, 6)
			for i := range orders {
				orders[i] = harness.Order(defs.OrderType_BuyMercenary, nil, harness.BuildKey(11, 0, 0, 0, 0), false)
			}
			g.Submit(p, orders...)
			g.EndTurn()
			state := g.Self(p)
			if len(state.Units) != 1+c.hired || state.Resources.Gold != float64(500-91*c.hired) {
				t.Errorf("batch: units %d, gold %v; want %d and %d", len(state.Units), state.Resources.Gold, 1+c.hired, 500-91*c.hired)
			}
			failures := state.Refusals()
			if len(failures) != 6-c.hired {
				t.Fatalf("failures %v, want %d", failures, 6-c.hired)
			}
			for _, reason := range failures {
				if reason != c.reason {
					t.Errorf("failure %q, want %q", reason, c.reason)
				}
			}
		})
	}
}

func TestMercenaryPlacementFailureRefundsReservation(t *testing.T) {
	for name, c := range map[string]struct {
		building uint32
		reason   string
	}{"ownership": {3, "zone lost ownership before placement"}, "population": {2, "population capped before placement"}} {
		t.Run(name, func(t *testing.T) {
			g := economyGame(t, `{"gold":500}`, `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":3,"player":0},"units":[{"id":1,"player":0,"count":1}]}]},{"x":1,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":2,"player":0}}]},`+remoteVillager, mercenaryRules)
			p := g.Human(0)
			var target uint64
			for _, b := range g.Self(p).Buildings {
				if b.BuildingID == c.building {
					target = b.InternalID
				}
			}
			g.Submit(p, harness.Order(defs.OrderType_BuyMercenary, nil, harness.BuildKey(11, 0, 0, 0, 0), false), harness.Order(defs.OrderType_BuildingDelete, []uint64{target}, 0, false))
			g.EndTurn()
			state := g.Self(p)
			if state.Resources.Gold != 500 || len(state.Units) != 1 || len(state.Refusals()) != 1 || state.Refusals()[0] != c.reason {
				t.Fatalf("placement did not refund without spawning: %+v, gold %v", state, state.Resources.Gold)
			}
			g.EndTurn()
			if g.Self(p).Resources.Gold != 500 || len(g.Self(p).Units) != 1 {
				t.Error("placement refund or spawn repeated")
			}
		})
	}
}

func TestMercenaryRestrictionsDoNotSpendGold(t *testing.T) {
	for name, c := range map[string]struct {
		workers, x    int
		extra, reason string
	}{
		"enemy territory": {1, -2, "", "space or zone not owned"},
		"unowned region":  {1, 0, `,"mercenaries_need_region":true,"regions":[{"name":"Split","spaces":[[0,0],[-2,0]]}]`, "region not owned"},
		"population cap":  {5, 0, "", "population capped"},
	} {
		t.Run(name, func(t *testing.T) {
			g := productionGame(t, c.workers, 1, mercenaryRules+c.extra)
			if name == "enemy territory" {
				g = economyGame(t, `{"gold":500}`, `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":1,"player":0},"units":[{"id":1,"player":0,"count":1}]}]},{"x":-2,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":11,"player":1,"count":1}]}]}`, mercenaryRules)
				owner := g.State(g.Human(1)).Space(-2, 0).Ownership
				if owner == nil || *owner != g.PlayerID(g.Human(1)) {
					t.Fatal("destination is not enemy territory")
				}
			}
			p := g.Human(0)
			g.Submit(p, harness.Order(defs.OrderType_BuyMercenary, nil, harness.BuildKey(11, c.x, 0, 0, 0), false))
			g.EndTurn()
			state := g.Self(p)
			if len(state.Units) != c.workers || state.Resources.Gold != 500 {
				t.Errorf("rejected hire: units %d, gold %v", len(state.Units), state.Resources.Gold)
			}
			if failures := state.Refusals(); len(failures) != 1 || failures[0] != c.reason {
				t.Errorf("failures %v, want %q", failures, c.reason)
			}
		})
	}
}

func TestResearchUnlocksMercenaryContracts(t *testing.T) {
	doc := `{"board_size":2,"players":2,"space_gold_income":0,"starting_bank":{"wood":300,"gold":500},"spaces":[{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":23,"player":0},"units":[{"id":1,"player":0,"count":1}]}]},{"x":-2,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":1,"player":1,"count":1}]}]}]}`
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"contracts": doc})
	g := startGame(t, "custom:contracts", 1, 2)
	p := g.Human(0)
	if len(g.Self(p).AvailableMercenaries) != 0 {
		t.Fatal("mercenaries available before contracts research")
	}
	b := g.Self(p).Buildings[0]
	g.Submit(p, harness.Order(defs.OrderType_BuildingResearch, []uint64{b.InternalID}, 4, false))
	g.EndTurn()
	turns := (defs.TechConfigs[4].Research_stamina + defs.BuildingConfigs[23].Turn_stamina - 1) / defs.BuildingConfigs[23].Turn_stamina
	for turn := 0; turn < turns && !g.Self(p).ResearchedTechs[4]; turn++ {
		g.EndTurn()
	}
	state := g.Self(p)
	if !state.ResearchedTechs[4] || state.Resources.Wood != 150 || len(state.AvailableMercenaries) != 2 {
		t.Fatalf("contracts research did not unlock both infantry with one payment: %+v", state)
	}
	if state.AvailableMercenaries[0].ID != 11 || state.AvailableMercenaries[1].ID != 12 {
		t.Fatalf("unlocked wrong mercenaries: %v", state.AvailableMercenaries)
	}
	g.Submit(p, harness.Order(defs.OrderType_BuyMercenary, nil, harness.BuildKey(12, 0, 0, 0, 0), false))
	g.EndTurn()
	state = g.Self(p)
	if len(state.Units) != 2 || state.Resources.Gold != 357 || state.Resources.Wood != 150 {
		t.Errorf("unlocked hiring: units %d, resources %+v", len(state.Units), state.Resources)
	}
}
