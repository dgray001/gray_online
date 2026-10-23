package combat

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
	"github.com/gin-gonic/gin"
)

func TestEliminationAndWinCondition(t *testing.T) {
	doc := `{"board_size":1,"players":2,"spaces":[{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":22,"player":0,"count":1},{"id":11,"player":1,"count":1}]}]}]}`
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"combat": doc})
	g := harness.NewConnectedGame(t, "custom:combat", 1, 2, 90) // 22 is Phalanx, 11 is Worker
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
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"combat": `{"board_size":1,"players":2,"spaces":[{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[]}]}]}`})
	g := harness.NewConnectedGame(t, "custom:combat", 1, 2, 90)
	if !g.Base.GameEnded() {
		t.Fatal("game with no survivors did not end")
	}
	checkOutcome(t, g.Risq.ToFrontend(0, true), []int{})
	updates := g.Base.ToFrontend(0, true)["viewer_updates"].([]gin.H)
	checkOutcome(t, updates[len(updates)-1]["content"].(gin.H)["game"].(gin.H), []int{})
}

func TestKillCredit(t *testing.T) {
	g := combatGame(t, 22, 11)
	p0, p1 := g.Human(0), g.Human(1)
	u0 := g.Self(p0).Units[0]
	u1 := g.Self(p1).Units[0]
	g.Submit(p0, harness.OrderAttackUnit([]uint64{u0.InternalID}, u1.InternalID))
	g.Submit(p1) // do nothing

	r0 := g.Risq.Results().Players[g.PlayerID(p0)]
	r1 := g.Risq.Results().Players[g.PlayerID(p1)]
	if r0.Kills != 1 {
		t.Errorf("p0 should have 1 kill, got %v", r0.Kills)
	}
	if r1.UnitsLost != 1 {
		t.Errorf("p1 should have 1 unit lost, got %v", r1.UnitsLost)
	}
}
