package combat

import (
	"fmt"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
	"testing"
)

func defeatBehaviorGame(t *testing.T, building uint32) *harness.Game {
	t.Helper()
	doc := fmt.Sprintf(`{"board_size":3,"players":3,"starting_bank":{"food":1000,"wood":1000,"stone":1000,"gold":1000},"space_gold_income":0,"spaces":[{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":%d,"player":0},"units":[{"id":1,"player":0,"count":1},{"id":13,"player":1,"count":1},{"id":1,"player":2,"count":1}]}]},{"x":-1,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":1,"player":1},"units":[{"id":12,"player":1,"count":1}]}]},{"x":1,"y":0,"terrain":1},{"x":2,"y":0,"terrain":1}]}`, building)
	fakeboard.UseConfig(t, shippedConfig, nil, map[string]string{"defeat-behavior": doc})
	g := harness.NewGame(t, "custom:defeat-behavior", 1, 3)
	for _, unit := range g.Self(g.Human(1)).Units {
		makePassive(g, 1, unit)
	}
	return g
}

func defeatBuilding(g *harness.Game, slot int, id uint32) harness.Building {
	g.T.Helper()
	for _, building := range g.Self(g.Human(slot)).Buildings {
		if building.BuildingID == id {
			return building
		}
	}
	g.T.Fatalf("missing building %d for slot %d", id, slot)
	return harness.Building{}
}
