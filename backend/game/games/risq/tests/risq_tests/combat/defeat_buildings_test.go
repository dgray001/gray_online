package combat

import (
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

func TestDefeatWithRemainingBuildings(t *testing.T) {
	for _, id := range []uint32{1, 2, 3, 11, 21, 22} {
		t.Run(fmt.Sprint(id), func(t *testing.T) {
			doc := fmt.Sprintf(`{"board_size":1,"players":2,"spaces":[{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":%d,"player":0},"units":[{"id":1,"player":1,"count":1}]}]}]}`, id)
			fakeboard.UseConfig(t, shippedConfig, nil, map[string]string{"defeat": doc})
			g := harness.NewGame(t, "custom:defeat", 1, 2)
			p := g.Self(g.Human(0))
			defeated := id != 1 && id != 22
			if p.Eliminated != defeated || g.Base.GameEnded() != defeated || len(p.Buildings) != 1 {
				t.Fatalf("building %d: defeated=%v ended=%v buildings=%d", id, p.Eliminated, g.Base.GameEnded(), len(p.Buildings))
			}
		})
	}
}
