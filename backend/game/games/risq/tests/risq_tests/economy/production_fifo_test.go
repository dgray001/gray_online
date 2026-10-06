package economy

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestProductionQueuePreservesDistinctItemOrder(t *testing.T) {
	g := economyGame(t, `{"food":300,"wood":100}`, `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":22,"player":0},"units":[{"id":1,"player":0,"count":1}]}]},`+remoteVillager, `,"unlimited_population":true`)
	p := g.Human(0)
	b := g.Self(p).Buildings[0].InternalID
	g.Submit(p, harness.OrderProduce([]uint64{b}, 11), harness.OrderProduce([]uint64{b}, 12))
	for turn := 1; turn <= 4; turn++ {
		g.EndTurn()
		state := g.Self(p)
		counts := map[uint32]int{}
		for _, u := range state.Units {
			counts[u.UnitID]++
		}
		wantPiercing := 0
		if turn >= 3 {
			wantPiercing = 1
		}
		if counts[1] != 1 || counts[11] != 1 || counts[12] != wantPiercing || state.Resources.Food != 180 || state.Resources.Wood != 40 {
			t.Errorf("turn %d: units %v, resources %+v", turn, counts, state.Resources)
		}
		queue := state.Buildings[0].ProductionQueue
		if turn < 3 {
			if len(queue) != 1 || queue[0].ItemID != 12 || queue[0].StaminaRemaining != 24-10*turn {
				t.Errorf("turn %d: queue %v, want piercing infantry with %d stamina remaining", turn, queue, 24-10*turn)
			}
		} else if len(queue) != 0 {
			t.Errorf("completed production retained queue %v", queue)
		}
	}
}
