package orders

import (
	"testing"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestActionsRequireGivingOrders(t *testing.T) {
	useOrderConfig(t, richBank, "")
	g := harness.NewUnstartedGame(t, "custom:orders", 1, 2)
	p := g.Human(0)
	payloads := blockedActions(g, p)
	payloads["unsubmit-orders"] = nil
	for kind, payload := range payloads {
		rejectAction(g, p, kind, payload, "Not currently giving orders")
	}
}

func TestUnknownAndEndedActionsHaveNoEffect(t *testing.T) {
	g := orderGame(t, richBank, "")
	p := g.Human(0)
	before := snapshotJSON(t, g.Risq.ToFrontend(uint64(p+1), false))
	g.Risq.PlayerAction(game.PlayerAction{Client_id: 999999, Kind: "submit-orders"})
	rawAction(g, p, "unknown-action", nil)
	if snapshotJSON(t, g.Risq.ToFrontend(uint64(p+1), false)) != before {
		t.Fatal("unknown action changed state")
	}
	g.Base.EndGame("test")
	before = snapshotJSON(t, g.Risq.ToFrontend(uint64(p+1), false))
	queued := queuedUpdates(g)
	for kind, payload := range blockedActions(g, p) {
		rawAction(g, p, kind, payload)
	}
	rawAction(g, p, "unsubmit-orders", nil)
	if snapshotJSON(t, g.Risq.ToFrontend(uint64(p+1), false)) != before || len(g.Failed(p)) != 0 || queuedUpdates(g) != queued {
		t.Fatal("unknown or ended action changed player")
	}
}
