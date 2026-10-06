package bridge

import (
	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"github.com/gin-gonic/gin"
	"testing"
)

func TestRunnerStopsWaitingAndSending(t *testing.T) {
	for _, blocked := range []bool{false, true} {
		t.Run(map[bool]string{false: "waiting", true: "sending"}[blocked], func(t *testing.T) {
			g, player := bridgeGame(t, ownScout+`,{"x":3,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":1,"player":1,"count":1}]}]}`)
			seen := &recordingModel{Model: ai.NoopModel{}, Seen: make(chan observation, 1)}
			w := runWorker(t, player, seen, make(chan game.PlayerAction))
			if blocked {
				payload := g.Risq.ToFrontend(uint64(g.Human(0)+1), false)
				player.AiUpdates <- &game.UpdateMessage{Kind: "start-turn", Content: gin.H{"game": payload}}
				receive(t, seen.Seen)
			}
			w.close(t)
			select {
			case action := <-w.Actions:
				t.Fatalf("post-stop action=%+v", action)
			default:
			}
		})
	}
}
