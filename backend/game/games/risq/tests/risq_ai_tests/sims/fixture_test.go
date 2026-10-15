package sims

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/config"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/aisim"
)

var mapFile string

func TestMain(m *testing.M) {
	cleanup := setupConfigDir()
	defer cleanup()
	os.Exit(m.Run())
}

func setupConfigDir() func() {
	dir, _ := os.MkdirTemp("", "sims-*")
	mapDir := filepath.Join(dir, "maps", "custom")
	_ = os.MkdirAll(mapDir, 0o755)
	mapFile = filepath.Join(mapDir, "ai-test.json")
	copyFiles(dir)
	_ = defs.LoadConfig(dir)
	return func() { _ = defs.LoadConfig(testconfig.Dir()); _ = os.RemoveAll(dir) }
}

func copyFiles(dir string) {
	files := []string{"ai/default.json", "bonuses.json", "buildings.json", "region_names.json", "resources.json", "techs.json", "terrains.json", "units.json"}
	for _, f := range files {
		data, _ := os.ReadFile(filepath.Join(testconfig.Dir(), f))
		dst := filepath.Join(dir, f)
		_ = os.MkdirAll(filepath.Dir(dst), 0o755)
		_ = os.WriteFile(dst, data, 0o644)
	}
}

const emptyModel = `{"rules":[]}`
const remote = `{"x":-3,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":1,"player":1},"units":[{"id":1,"player":1,"count":1}]}]}`

func once(actions string) string {
	return `{"rules":[{"when":{"turn_equals":{"amount":1}},"then":[` + actions + `]}]}`
}

func fixture(t *testing.T, bank, spaces, extra string, models ...string) *aisim.Game {
	t.Helper()
	document := fmt.Sprintf(`{"board_size":4,"players":%d,"space_gold_income":0,"starting_bank":%s,"spaces":[%s]%s}`, len(models), bank, spaces, extra)
	if err := os.WriteFile(mapFile, []byte(document), 0o644); err != nil {
		t.Fatal(err)
	}
	return aisim.New(t, "custom:ai-test", models...)
}

func TestEmptyModelsResolveTurnsAndStop(t *testing.T) {
	g := fixture(t, `{}`, `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":2,"player":0},"units":[{"id":1,"player":0,"count":1}]}]},`+remote, "", emptyModel, emptyModel)
	g.Run(t, 3)
	if len(g.Trace) != 6 {
		t.Fatalf("actions=%+v", g.Trace)
	}
	g.Stop(t)
	g.Stop(t)
}

func assertOrderCount(t *testing.T, g *aisim.Game, slot int, kind defs.OrderType, want int) {
	t.Helper()
	count := 0
	for _, action := range g.Trace {
		if action.Kind != "submit-orders" || action.Ai_id != int(g.Player(t, slot).GetAiId()) {
			continue
		}
		for _, order := range action.Action["orders"].([]defs.OrderFromFrontend) {
			if order.Player_id != slot {
				t.Fatalf("wrong player: %+v", order)
			}
			if order.Order_type == uint8(kind) {
				count++
			}
		}
	}
	if count != want {
		t.Fatalf("slot %d kind %d count %d, want %d; trace=%+v", slot, kind, count, want, g.Trace)
	}
}

func assertSubmissions(t *testing.T, g *aisim.Game, slot, want int) {
	t.Helper()
	count := 0
	owned := make(map[uint64]bool)
	for _, unit := range g.Own(t, slot).Units {
		owned[unit.InternalID] = unit.PlayerID == slot
	}
	for _, action := range g.Trace {
		if action.Kind != "submit-orders" || action.Ai_id != int(g.Player(t, slot).GetAiId()) {
			continue
		}
		count++
		for _, order := range action.Action["orders"].([]defs.OrderFromFrontend) {
			for _, id := range order.Subjects {
				if !owned[id] {
					t.Fatalf("foreign subject: %+v", order)
				}
			}
		}
	}
	if count != want {
		t.Fatalf("slot %d submissions %d, want %d", slot, count, want)
	}
}

func TestThreePlayersKeepDecisionsSeparate(t *testing.T) {
	food := once(`{"action":"gather","category":"food","max":1}`)
	wood := once(`{"action":"gather","category":"wood","max":1}`)
	spaces := `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":1,"player":0,"count":1}],"resource":1},{"x":1,"y":0,"building":{"id":2,"player":0}}]},` + remote + `,{"x":3,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":1,"player":2,"count":1}],"resource":11},{"x":1,"y":0,"building":{"id":2,"player":2}}]}`
	g := fixture(t, `{}`, spaces, "", food, emptyModel, wood)
	g.Run(t, 2)
	assertOrderCount(t, g, 0, defs.OrderType_UnitGather, 1)
	assertOrderCount(t, g, 2, defs.OrderType_UnitGather, 1)
	if g.Own(t, 0).Resources.Food <= 0 || g.Own(t, 0).Resources.Wood != 0 || g.Own(t, 2).Resources.Wood <= 0 || g.Own(t, 2).Resources.Food != 0 {
		t.Fatal("cross-player resource accounting")
	}
	for slot := range 3 {
		assertSubmissions(t, g, slot, 2)
	}
}

const duel = `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":13,"player":0,"count":1},{"id":1,"player":1,"count":1}]},{"x":1,"y":0,"building":{"id":2,"player":0}},{"x":-1,"y":0,"building":{"id":2,"player":1}}]}`

func TestGameEndStopsAllAIWorkers(t *testing.T) {
	model := once(`{"action":"set_unit_behavior","stance":"passive","attack_back":false},{"action":"attack","target":"economic","max":1}`)
	g := fixture(t, `{}`, duel, "", model, emptyModel)
	g.Run(t, 1)
	if !g.Base.GameEnded() || !g.Own(t, 1).Eliminated || g.Own(t, 0).Eliminated {
		t.Fatalf("end state=%+v", g.Risq.Results())
	}
	g.Stop(t)
	g.Stop(t)
	select {
	case <-g.Base.GameEndedChannel:
	default:
		t.Fatal("missing game-end notification")
	}
}

func TestEliminatedAIStopsWhileSurvivorsContinue(t *testing.T) {
	model := once(`{"action":"set_unit_behavior","stance":"passive","attack_back":false},{"action":"attack","targets":"enemy_units","score":1,"max":1}`)
	third := `{"x":3,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":1,"player":2},"units":[{"id":1,"player":2,"count":1}]}]}`
	g := fixture(t, `{}`, duel+","+third, "", model, emptyModel, emptyModel)
	g.Run(t, 2)
	if g.Base.GameEnded() || !g.Own(t, 1).Eliminated {
		t.Fatalf("elimination=%+v", g.Risq.Results())
	}
	assertSubmissions(t, g, 1, 1)
	assertSubmissions(t, g, 2, 2)
	g.Stop(t)
}

func TestResultsMatchResolvedState(t *testing.T) {
	spaces := `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"resource":1,"units":[{"id":1,"player":0,"count":1}]},{"x":1,"y":0,"building":{"id":2,"player":0}}]},` + remote
	g := fixture(t, `{}`, spaces, "", once(`{"action":"gather","category":"food"}`), emptyModel)
	g.Run(t, 3)
	result := g.Risq.Results()
	if result.TurnNumber != g.Risq.TurnNumber() || len(result.Players) != 2 {
		t.Fatalf("results=%+v", result)
	}
	seen := make(map[int]bool)
	for _, player := range result.Players {
		own := g.Own(t, player.PlayerId)
		if seen[player.PlayerId] || player.Nickname != g.Player(t, player.PlayerId).GetNickname() {
			t.Fatalf("identity=%+v", player)
		}
		seen[player.PlayerId] = true
		if player.Units != len(own.Units) || player.Buildings != len(own.Buildings) || player.Eliminated != own.Eliminated {
			t.Fatalf("counts=%+v own=%+v", player, own)
		}
		land, techs := 0, 0
		for _, row := range aisim.Decode(t, aisim.Payload(t, g.Player(t, player.PlayerId))).Spaces {
			for _, space := range row {
				if space != nil && space.Ownership != nil && *space.Ownership == player.PlayerId {
					land++
				}
			}
		}
		for _, researched := range own.ResearchedTechs {
			if researched {
				techs++
			}
		}
		if player.Land != land || player.TechsResearched != techs {
			t.Fatalf("land/techs=%+v", player)
		}
		if player.Gathered.Food != own.Resources.Food || player.Gathered.Wood != 0 || player.Gathered.Stone != 0 || player.Gathered.Gold != 0 {
			t.Fatalf("gathered=%+v stockpile=%+v", player.Gathered, own.Resources)
		}
		for _, value := range []float64{player.Economy.VillagerIdlePct, player.Economy.VillagerIdlePctEarly, player.Economy.VillageCenterIdlePct, player.Economy.VillageCenterIdlePctEarly} {
			if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 100 {
				t.Fatalf("idle percentage=%v", value)
			}
		}
	}
}
