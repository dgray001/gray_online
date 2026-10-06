package economy

import (
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

func gatherGame(t *testing.T, resource uint32, contested bool) *harness.Game {
	t.Helper()
	enemy := `,{"id":1,"player":1,"count":1}`
	if !contested {
		enemy = ""
	}
	doc := fmt.Sprintf(`{"board_size":2,"players":2,"spaces":[{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"resource":%d,"units":[{"id":1,"player":0,"count":1}%s]}]},{"x":2,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":1,"player":1,"count":1}]}]}]}`, resource, enemy)
	fakeboard.UseConfig(t, shippedConfig, nil, map[string]string{"gather": doc})
	return harness.NewGame(t, "custom:gather", 1, 2)
}

func balance(g *harness.Game, human int, category defs.RisqResourceCategory) float64 {
	r := g.Self(human).Resources
	return []float64{r.Food, r.Wood, r.Stone, r.Gold}[category.Index()]
}

func deplete(g *harness.Game, human int) {
	g.T.Helper()
	for turn := 0; turn < 60 && len(g.State(human).Space(0, 0).Resources) > 0; turn++ {
		g.EndTurn()
	}
	space := g.State(human).Space(0, 0)
	if len(space.Resources) != 0 {
		g.T.Fatal("resource node not depleted within 60 turns")
	}
	for _, row := range space.Zones {
		for _, zone := range row {
			if zone.Resource != nil {
				g.T.Fatal("depleted node remains in zone payload")
			}
		}
	}
}

func submitGather(g *harness.Game, human int) {
	g.T.Helper()
	for _, u := range g.Self(human).Units {
		if u.Space == (harness.Coord{}) && u.Zone == (harness.Coord{}) {
			g.Submit(human, harness.OrderGather([]uint64{u.InternalID}, 0, 0, 0, 0))
			return
		}
	}
	g.T.Fatal("no villager on the resource zone")
}
