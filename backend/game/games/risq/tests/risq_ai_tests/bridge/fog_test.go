package bridge

import (
	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
	"testing"
)

func TestBridgeKeepsLastKnownBuildingsAfterCoverageIsLost(t *testing.T) {
	spaces := `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":1,"player":0,"count":1}]},{"x":1,"y":0,"building":{"id":2,"player":1}}]},{"x":-3,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":1,"player":0,"count":1}]}]},{"x":3,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":1,"player":1,"count":1}]}]}`
	g, player := bridgeGame(t, spaces)
	seen := &recordingModel{Model: ai.NoopModel{}, Seen: make(chan observation, 1)}
	w := runWorker(t, player, seen, make(chan game.PlayerAction, 4))
	owner, enemy := g.Human(0), g.Human(1)
	player.AiUpdates <- &game.UpdateMessage{Kind: "start-turn", Content: gin.H{"game": g.Risq.ToFrontend(uint64(owner+1), false)}}
	initial := receive(t, seen.Seen)
	receive(t, w.Actions)
	if len(initial.VisibleBuildings) != 1 {
		t.Fatalf("initial buildings=%+v", initial)
	}
	var scout uint64
	for _, unit := range g.Self(owner).Units {
		if unit.Space.X == 0 {
			scout = unit.InternalID
		}
	}
	if scout == 0 {
		t.Fatal("missing scout")
	}
	g.Submit(owner, harness.Order(defs.OrderType_UnitDelete, []uint64{scout}, 0, false))
	g.EndTurn()
	g.Submit(enemy, harness.Order(defs.OrderType_BuildingDelete, []uint64{g.Self(enemy).Buildings[0].InternalID}, 0, false))
	g.EndTurn()
	g.EndTurn()
	if g.State(owner).Space(0, 0).Visibility != 1 {
		t.Fatal("fixture never lost coverage")
	}
	player.AiUpdates <- &game.UpdateMessage{Kind: "start-turn", Content: gin.H{"game": g.Risq.ToFrontend(uint64(owner+1), false)}}
	fog := receive(t, seen.Seen)
	receive(t, w.Actions)
	if len(fog.VisibleBuildings) != 0 || len(fog.Units) != 0 || len(fog.KnownBuildings) != 1 || fog.KnownBuildings[0].InternalID != initial.KnownBuildings[0].InternalID {
		t.Fatalf("fog=%+v", fog)
	}
}
