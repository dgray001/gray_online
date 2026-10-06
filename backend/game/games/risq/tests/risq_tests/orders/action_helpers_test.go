package orders

import (
	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

func rejectAction(g *harness.Game, p int, kind string, action gin.H, message string) string {
	g.T.Helper()
	before := snapshotJSON(g.T, g.Risq.ToFrontend(uint64(p+1), false))
	queued := queuedUpdates(g)
	g.Risq.PlayerAction(game.PlayerAction{Client_id: p + 1, Kind: kind, Action: action})
	failures := g.Failed(p)
	if len(failures) != 1 || (message != "" && failures[0] != message) {
		g.T.Fatalf("%s failures %v, want %q", kind, failures, message)
	}
	if after := snapshotJSON(g.T, g.Risq.ToFrontend(uint64(p+1), false)); before != after {
		g.T.Fatalf("rejected %s changed serialized state: before %s, after %s", kind, before, after)
	}
	if after := queuedUpdates(g); after != queued {
		g.T.Fatalf("rejected %s emitted a success notification", kind)
	}
	return failures[0]
}
