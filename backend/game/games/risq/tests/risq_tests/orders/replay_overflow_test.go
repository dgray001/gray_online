package orders

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/gin-gonic/gin"
)

func TestDroppedPlayerWakeupRemainsReplayable(t *testing.T) {
	g := orderGame(t, richBank, "")
	p := g.Human(0)
	player := g.Base.Players[uint64(p+1)]
	latestUpdate(t, player.Updates)
	for range cap(player.Updates) {
		rawAction(g, p, "set-unit-behavior", gin.H{"internal_ids": unitIDs(g, p, 11), "stance": 1})
	}
	rawAction(g, p, "set-unit-behavior", gin.H{"internal_ids": unitIDs(g, p, 11), "stance": uint8(defs.UnitStance_STAND_GROUND)})
	for len(player.Updates) > 0 {
		if u := nextUpdate(t, player.Updates); u.Content["stance"] != uint8(1) {
			t.Fatal("full buffer unexpectedly accepted newest wakeup")
		}
	}
	g.Base.ResendLastUpdate(uint64(p + 1))
	if u := nextUpdate(t, player.Updates); u.Id != 1 || u.Kind != "unit-behavior-set" || u.Content["stance"] != uint8(defs.UnitStance_STAND_GROUND) {
		t.Fatalf("dropped update was not replayed: %+v", u)
	}
}
