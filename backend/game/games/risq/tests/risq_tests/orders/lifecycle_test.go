package orders

import (
	"os"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/aisim"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

func TestEliminatedPlayersCannotSubmitAndDoNotBlockTurns(t *testing.T) {
	g := newGame(t, layout{humans: 3, spaces: `,{"x":-2,"y":2,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":1,"player":2,"count":1}]}]}`})
	p := g.Human(0)
	var orders []defs.OrderFromFrontend
	state := g.Self(p)
	for _, u := range state.Units {
		orders = append(orders, harness.Order(defs.OrderType_UnitDelete, []uint64{u.InternalID}, 0, true))
	}
	for _, b := range state.Buildings {
		orders = append(orders, harness.Order(defs.OrderType_BuildingDelete, []uint64{b.InternalID}, 0, true))
	}
	g.Submit(p, orders...)
	g.EndTurn()
	if !g.Self(p).Eliminated || g.Base.GameEnded() {
		t.Fatal("setup did not leave an eliminated player and two survivors")
	}
	g.SubmitRejected(p, harness.OrderMove(nil, 1, 0), "Eliminated players cannot submit orders")
	turn := g.Risq.TurnNumber()
	g.Submit(g.Human(1))
	g.Submit(g.Human(2))
	if g.Risq.TurnNumber() != turn+1 {
		t.Fatal("eliminated player blocked turn resolution")
	}
}

func defeatMap(t *testing.T) {
	t.Helper()
	doc, err := os.ReadFile(testConfig + "/maps/custom/defeat_test.json")
	if err != nil {
		t.Fatal(err)
	}
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"defeat_test": string(doc)})
}

func TestDefeatMapStartsWithRedoubtAndEnemySoldier(t *testing.T) {
	defeatMap(t)
	g := harness.NewGame(t, "custom:defeat_test", 1, 2)
	p := g.Human(0)
	mine, enemy := g.Self(p), g.Self(g.Human(1))
	if len(mine.Units) != 0 || len(mine.Buildings) != 1 || mine.Buildings[0].BuildingID != 23 {
		t.Fatal("human player should have only a redoubt")
	}
	if len(enemy.Buildings) != 0 || len(enemy.Units) != 1 || enemy.Units[0].UnitID != 103 || enemy.Units[0].Space != mine.Buildings[0].Space {
		t.Fatal("enemy should have one heavy infantry in the redoubt's space")
	}
}

func TestTemporaryRetreatAILeavesRedoubtRange(t *testing.T) {
	defeatMap(t)
	model, err := os.ReadFile(testConfig + "/ai/temp_retreat.json")
	if err != nil {
		t.Fatal(err)
	}
	g := aisim.New(t, "custom:defeat_test", `{"rules":[]}`, string(model))
	g.Run(t, 4)
	enemy := g.Own(t, 1)
	if len(enemy.Units) != 1 || enemy.Units[0].Space != (harness.Coord{X: 1}) || len(enemy.ActiveOrders) != 0 {
		t.Fatalf("retreat AI did not move and stop: %+v", enemy)
	}
}
