package bridge

import (
	"fmt"
	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/gin-gonic/gin"
	"reflect"
	"sort"
	"testing"
)

func TestUnseenChangesDoNotChangeViewOrDecision(t *testing.T) {
	var observations []observation
	for count := 1; count <= 2; count++ {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			spaces := ownScout + fmt.Sprintf(`,{"x":3,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"resource":%d,"units":[{"id":1,"player":1,"count":%d}]},{"x":1,"y":0,"building":{"id":2,"player":1}}]}`, count, count)
			g, player := bridgeGame(t, spaces)
			seen := &recordingModel{Model: parsedModel(t, `{"action":"attack","targets":"enemy_units","score":1}`), Seen: make(chan observation, 1)}
			w := runWorker(t, player, seen, make(chan game.PlayerAction, 4))
			player.AiUpdates <- &game.UpdateMessage{Kind: "start-turn", Content: gin.H{"game": g.Risq.ToFrontend(uint64(g.Human(0)+1), false)}}
			view := receive(t, seen.Seen)
			orders := receive(t, w.Actions).Action["orders"].([]defs.OrderFromFrontend)
			if len(orders) != 0 {
				t.Fatalf("hidden target orders=%+v", orders)
			}
			sort.Slice(view.Spaces, func(i, j int) bool {
				a, b := view.Spaces[i].Space, view.Spaces[j].Space
				return a.X < b.X || a.X == b.X && a.Y < b.Y
			})
			observations = append(observations, view)
		})
	}
	if len(observations) != 2 || !reflect.DeepEqual(observations[0], observations[1]) {
		t.Fatalf("hidden changes leaked: %+v", observations)
	}
}
