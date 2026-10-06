package combat

import (
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
	"testing"
)

func TestEliminationAndWinCondition(t *testing.T) {
	g := combatGame(t, 13, 1) // 13 is Heavy Infantry (50hp, 9att), 1 is Villager (8hp, 5att)
	p0, p1 := g.Human(0), g.Human(1)
	if _, exists := g.Risq.ToFrontend(0, true)["outcome"]; exists {
		t.Fatal("ongoing game has an outcome")
	}
	u0 := g.Self(p0).Units[0]
	u1 := g.Self(p1).Units[0]
	g.Submit(p0, harness.OrderAttackUnit([]uint64{u0.InternalID}, u1.InternalID))
	g.Submit(p1) // Do nothing

	state := g.State(p0)
	if !state.Player(g.PlayerID(p1)).Eliminated {
		t.Errorf("p1 should be eliminated after losing all units and buildings")
	}
	if !g.Base.GameEnded() {
		t.Errorf("game should be ended when only one player remains")
	}
	winners := []int{g.PlayerID(p0)}
	for _, player := range g.Base.Players {
		checkOutcome(t, g.Risq.ToFrontend(player.GetClientId(), false), winners)
		updates := player.ToFrontend(true)["updates"].([]gin.H)
		checkOutcome(t, updates[len(updates)-1]["content"].(gin.H)["game"].(gin.H), winners)
	}
	checkOutcome(t, g.Risq.ToFrontend(0, true), winners)
	updates := g.Base.ToFrontend(0, true)["viewer_updates"].([]gin.H)
	checkOutcome(t, updates[len(updates)-1]["content"].(gin.H)["game"].(gin.H), winners)
}

func TestNoSurvivorsOutcome(t *testing.T) {
	g := combatGame(t, 0, 0)
	if !g.Base.GameEnded() {
		t.Fatal("game with no survivors did not end")
	}
	checkOutcome(t, g.Risq.ToFrontend(0, true), []int{})
	updates := g.Base.ToFrontend(0, true)["viewer_updates"].([]gin.H)
	checkOutcome(t, updates[len(updates)-1]["content"].(gin.H)["game"].(gin.H), []int{})
}
