package orders

import (
	"os"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/aisim"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

func TestDefeatMapStartsWithRedoubtAndEnemySoldier(t *testing.T) {
	doc, err := os.ReadFile(testConfig + "/maps/custom/defeat_test.json")
	if err != nil {
		t.Fatal(err)
	}
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"defeat_test": string(doc)})
	g := harness.NewGame(t, "custom:defeat_test", 1, 2)
	p := g.Human(0)
	if len(g.Self(p).Units) != 0 || len(g.Self(p).Buildings) != 1 || g.Self(p).Buildings[0].BuildingID != 23 {
		t.Fatal("human player should have only a redoubt")
	}
	enemy := g.Self(g.Human(1))
	if len(enemy.Buildings) != 0 || len(enemy.Units) != 1 || enemy.Units[0].UnitID != 13 || enemy.Units[0].Space != g.Self(p).Buildings[0].Space {
		t.Fatal("enemy should have one heavy infantry in the redoubt's space")
	}
}

func TestTemporaryRetreatAILeavesRedoubtRange(t *testing.T) {
	doc, err := os.ReadFile(testConfig + "/maps/custom/defeat_test.json")
	if err != nil {
		t.Fatal(err)
	}
	model, err := os.ReadFile(testConfig + "/ai/temp_retreat.json")
	if err != nil {
		t.Fatal(err)
	}
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"defeat_test": string(doc)})
	g := aisim.New(t, "custom:defeat_test", `{"rules":[]}`, string(model))
	g.Run(t, 4)
	enemy := g.Own(t, 1)
	if len(enemy.Units) != 1 || enemy.Units[0].Space != (harness.Coord{X: 1}) || len(enemy.ActiveOrders) != 0 {
		t.Fatalf("retreat AI did not move and stop: %+v", enemy)
	}
}
