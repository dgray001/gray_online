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
	"github.com/gin-gonic/gin"
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
const remote = `{"x":-3,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":201,"player":1},"units":[{"id":11,"player":1,"count":1}]}]}`

// The hire rule checks for tech 4 by id, so these tests register a mercenary tech under that id that unlocks grunts
func registerMercenaryTech(t *testing.T) {
	t.Helper()
	defs.TechConfigs[4] = defs.TechConfig{Display_name: "Test Mercenary Tech", Research_stamina: 45, Unlocks_mercenary_ids: []uint32{21}}
	t.Cleanup(func() { delete(defs.TechConfigs, 4) })
}

func onTurn(turn int) string {
	return fmt.Sprintf(`{"value_equals":{"value":"gameVar(turn)","amount":%d}}`, turn)
}

func once(actions string) string {
	return `{"rules":[{"when":` + onTurn(1) + `,"then":[` + actions + `]}]}`
}

const (
	passiveUnits   = `{"action":"select","actors":{"kind":"units"},"issue":[{"set_behavior":{"stance":"passive","attack_back":false}}]}`
	quietBuildings = `{"action":"select","actors":{"kind":"buildings"},"issue":[{"set_behavior":{"auto_attack":false}}]}`
	attackEconomic = `{"action":"select","actors":{"kind":"units","where":{"if":"var(actor.idle)"},"limit":1},"pick":{"kind":"units","where":{"owner":"enemy","unit_types":["economic"]}},"issue":[{"attack":{}}]}`
	attackEnemy    = `{"action":"select","actors":{"kind":"units","where":{"if":"var(actor.idle)"},"limit":1},"pick":{"kind":"units","where":{"owner":"enemy"}},"issue":[{"attack":{}}]}`
	homeZone       = `{"kind":"zones","where":{"if":"(var(target.x) == 0) * (var(target.y) == 0) * (var(target.zone_x) == 0) * (var(target.zone_y) == 0)"}}`
)

// Sends idle villagers to a node of the category, never past the node's gather capacity counting gatherers already ordered
func gather(category string, limit int) string {
	actors := `{"kind":"units","where":{"if":"var(actor.idle)"}`
	if limit > 0 {
		actors += fmt.Sprintf(`,"limit":%d`, limit)
	}
	return fmt.Sprintf(`{"action":"select","actors":%s},"pick":{"kind":"gatherables","where":{"category":%q},"let":{"gatherers":{"count":{"kind":"orders","where":{"in":{"order_kind":["gather"]},"if":"var(item.target_uid) == var(target.uid)"}}}},"capacity":"var(target.gather_capacity) - var(target.gatherers)"},"issue":[{"gather":{}}]}`, actors, category)
}

// The create and hire verbs do not stop at the population cap or an empty stockpile, so the count says how many fit
func create(unit_id int, queue int) string {
	fits := fmt.Sprintf(`min(%d, gameVar(population_limit) - gameVar(population), round(gameVar(food) / config(unit, cost_food, %d) - 0.5))`, queue, unit_id)
	return fmt.Sprintf(`{"action":"select","actors":{"kind":"buildings","where":{"can_create":%d}},"issue":[{"create":{"unit_id":%d,"queue":%q}}]}`, unit_id, unit_id, fits)
}

func hire(unit_id int, count int) string {
	fits := fmt.Sprintf(`min(%d, gameVar(population_limit) - gameVar(population))`, count)
	return fmt.Sprintf(`{"action":"select","pick":%s,"issue":[{"hire":{"unit_id":%d,"count":%q}}]}`, homeZone, unit_id, fits)
}

func stop(order string) string {
	return fmt.Sprintf(`{"action":"select","actors":{"kind":"buildings"},"issue":[{"stop":{"orders":[%q]}}]}`, order)
}

// Builds the building on the first free zone of the origin space if the wood is there, so a second rule cannot spend it again
func build(building_id int) string {
	return fmt.Sprintf(`{"action":"select","actors":{"kind":"units","where":{"if":["var(actor.idle)","gameVar(wood) >= config(building, cost_wood, %d)"]},"limit":1},"pick":{"kind":"zones","where":{"if":"var(target.x) == 0"},"limit":1},"issue":[{"build":{"building_id":%d}}]}`, building_id, building_id)
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
	g := fixture(t, `{}`, `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":202,"player":0},"units":[{"id":11,"player":0,"count":1}]}]},`+remote, "", emptyModel, emptyModel)
	g.Run(t, 3)
	if len(g.Trace) != 6 {
		t.Fatalf("actions=%+v", g.Trace)
	}
	g.Stop(t)
	g.Stop(t)
}

func TestAiSnapshotsOmitReplayData(t *testing.T) {
	g := fixture(t, `{}`, `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":202,"player":0},"units":[{"id":11,"player":0,"count":1}]}]},`+remote, "", emptyModel, emptyModel)
	g.Run(t, 2)
	updates := g.Player(t, 0).ToFrontend(true)["updates"].([]gin.H)
	state := updates[len(updates)-1]["content"].(gin.H)["game"].(gin.H)
	if state["last_turn_replay"] != nil {
		t.Error("ai snapshot carries last_turn_replay")
	}
	if g.Risq.ToFrontend(0, true)["last_turn_replay"] != nil {
		t.Error("a game with no human or viewer recorded a journal")
	}
	for _, p := range state["players"].([]gin.H) {
		if _, ok := p["turn_report"]; ok {
			t.Error("ai snapshot carries a turn_report")
		}
		for _, unit := range p["units"].([]gin.H) {
			if _, ok := unit["tick_actions"]; ok {
				t.Error("ai snapshot carries unit tick_actions")
			}
		}
	}
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
	food := once(gather("food", 1))
	wood := once(gather("wood", 1))
	spaces := `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":11,"player":0,"count":1}],"resource":1},{"x":1,"y":0,"building":{"id":202,"player":0}}]},` + remote + `,{"x":3,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":11,"player":2,"count":1}],"resource":11},{"x":1,"y":0,"building":{"id":202,"player":2}}]}`
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

const duel = `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":22,"player":0,"count":1},{"id":11,"player":1,"count":1}]},{"x":1,"y":0,"building":{"id":202,"player":0}},{"x":-1,"y":0,"building":{"id":202,"player":1}}]}`

func TestGameEndStopsAllAIWorkers(t *testing.T) {
	model := once(passiveUnits + `,` + attackEconomic)
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
	model := once(passiveUnits + `,` + attackEnemy)
	third := `{"x":3,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":201,"player":2},"units":[{"id":11,"player":2,"count":1}]}]}`
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
	spaces := `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"resource":1,"units":[{"id":11,"player":0,"count":1}]},{"x":1,"y":0,"building":{"id":202,"player":0}}]},` + remote
	g := fixture(t, `{}`, spaces, "", once(gather("food", 0)), emptyModel)
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
