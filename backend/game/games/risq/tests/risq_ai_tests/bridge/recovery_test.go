package bridge

import (
	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/gin-gonic/gin"
	"testing"
)

func TestRunnerFallbackOncePerTurn(t *testing.T) {
	g, player := bridgeGame(t, ownScout+`,{"x":3,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":1,"player":1,"count":1}]}]}`)
	player.AiFailedUpdates = make(chan *game.UpdateMessage)
	w := runWorker(t, player, ai.NoopModel{}, make(chan game.PlayerAction, 8))
	update := &game.UpdateMessage{Kind: "start-turn", Content: gin.H{"game": g.Risq.ToFrontend(uint64(g.Human(0)+1), false)}}
	failure := &game.UpdateMessage{Kind: "submit-orders-failed", Content: gin.H{"message": "test rejection"}}
	player.AiUpdates <- update
	assertEmptySubmission(t, receive(t, w.Actions))
	player.AiFailedUpdates <- failure
	assertEmptySubmission(t, receive(t, w.Actions))
	player.AiFailedUpdates <- failure
	player.AiFailedUpdates <- failure
	player.AiUpdates <- update
	assertEmptySubmission(t, receive(t, w.Actions))
	player.AiFailedUpdates <- failure
	assertEmptySubmission(t, receive(t, w.Actions))
	w.close(t)
	if len(w.Actions) != 0 {
		t.Fatalf("extra fallback submissions: %d", len(w.Actions))
	}
}

func assertEmptySubmission(t *testing.T, action game.PlayerAction) {
	t.Helper()
	if action.Kind != "submit-orders" || action.Ai_id != 1 || action.Client_id != 0 || len(action.Action["orders"].([]defs.OrderFromFrontend)) != 0 {
		t.Fatalf("fallback=%+v", action)
	}
}
