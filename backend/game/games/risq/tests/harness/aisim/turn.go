package aisim

import (
	"testing"
	"time"
)

func (g *Game) Run(t *testing.T, turns uint16) {
	t.Helper()
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	for completed, _ := g.Risq.Standings(); completed < turns; completed, _ = g.Risq.Standings() {
		select {
		case action := <-g.Actions:
			g.Trace = append(g.Trace, action)
			g.Risq.PlayerAction(action)
		case <-g.Base.GameEndedChannel:
			t.Fatalf("game ended before turn %d: %+v", turns, g.Risq.Results())
		case <-deadline.C:
			t.Fatalf("waiting for turn %d: state=%+v trace=%+v", turns, g.State(t), g.Trace)
		}
		for len(g.Base.ViewerUpdates) > 0 {
			<-g.Base.ViewerUpdates
		}
		if finished, _ := g.Risq.Standings(); finished > completed {
			g.Check(t)
			for slot := range len(g.Base.AiPlayers) {
				player := g.Own(t, slot)
				if failures := player.Refusals(); len(failures) != 0 {
					t.Fatalf("player %d refused orders: %v; trace=%+v", player.Player.PlayerID, failures, g.Trace)
				}
			}
		}
	}
}
