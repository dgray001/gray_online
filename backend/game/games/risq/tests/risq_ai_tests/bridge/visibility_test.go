package bridge

import (
	"fmt"
	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
	"testing"
)

const ownScout = `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":11,"player":0,"count":1}]}]}`

func TestRealPayloadVisibility(t *testing.T) {
	for _, c := range []struct {
		name   string
		x      int
		vision uint8
	}{{"unexplored", 3, 0}, {"count", 1, 2}, {"full", 0, 3}, {"spy", 0, 4}} {
		t.Run(c.name, func(t *testing.T) {
			own := ""
			if c.x == 0 {
				own = `,{"id":11,"player":0,"count":1}`
			}
			spaces := fmt.Sprintf(`{"x":%d,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"resource":1,"units":[{"id":1,"player":1,"count":1}%s]},{"x":1,"y":0,"building":{"id":2,"player":1}}]}`, c.x, own)
			if c.x != 0 {
				spaces += "," + ownScout
			}
			g, player := bridgeGame(t, spaces, func() {
				if c.vision == 4 {
					config := defs.UnitConfigs[11]
					config.Vision.Space = defs.VisibilitySpy
					defs.UnitConfigs[11] = config
				}
			})
			g.Action(g.Human(0), "set-unit-behavior", gin.H{"internal_ids": []uint64{g.Self(g.Human(0)).Units[0].InternalID}, "stance": uint8(defs.UnitStance_PASSIVE), "attack_back": false})
			p := g.Human(1)
			g.Submit(p, harness.OrderGather([]uint64{g.Self(p).Units[0].InternalID}, c.x, 0, 0, 0))
			g.EndTurn()
			payload := g.Risq.ToFrontend(uint64(g.Human(0)+1), false)
			seen := &recordingModel{Model: parsedModel(t, `{"action":"attack","targets":"enemy_units","score":1,"max":1}`), Seen: make(chan observation, 1)}
			w := runWorker(t, player, seen, make(chan game.PlayerAction, 4))
			player.AiUpdates <- &game.UpdateMessage{Kind: "start-turn", Content: gin.H{"game": payload}}
			observed := receive(t, seen.Seen)
			action := receive(t, w.Actions)
			want := 0
			if c.vision >= 3 {
				want = 1
			}
			if orders := action.Action["orders"].([]defs.OrderFromFrontend); len(orders) != want {
				t.Fatalf("vision %d orders=%+v", c.vision, orders)
			}
			assertVisibility(t, observed, c.x, c.vision)
		})
	}
}
