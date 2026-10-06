package bridge

import (
	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"github.com/gin-gonic/gin"
	"testing"
)

func TestRunnerIgnoresOtherUpdatesAndSubmitsMalformedSnapshot(t *testing.T) {
	_, player := bridgeGame(t, ownScout+`,{"x":3,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":1,"player":1,"count":1}]}]}`)
	player.AiUpdates = make(chan *game.UpdateMessage)
	seen := &recordingModel{Model: ai.NoopModel{}, Seen: make(chan observation, 2)}
	w := runWorker(t, player, seen, make(chan game.PlayerAction, 8))
	player.AiUpdates <- &game.UpdateMessage{Kind: "submitted-orders"}
	player.AiUpdates <- &game.UpdateMessage{Kind: "start-turn", Content: gin.H{"game": gin.H{"turn_number": "bad"}}}
	assertEmptySubmission(t, receive(t, w.Actions))
	player.AiUpdates <- &game.UpdateMessage{Kind: "start-turn", Content: gin.H{"game": gin.H{}}}
	assertEmptySubmission(t, receive(t, w.Actions))
	w.close(t)
	if len(seen.Seen) != 0 || len(w.Actions) != 0 {
		t.Fatal("invalid or unrelated update reached the model")
	}
}
