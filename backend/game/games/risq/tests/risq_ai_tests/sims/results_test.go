package sims

import (
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/aisim"
	"math"
	"testing"
)

func TestResultsMatchResolvedState(t *testing.T) {
	spaces := `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"resource":1,"units":[{"id":1,"player":0,"count":1}]},{"x":1,"y":0,"building":{"id":2,"player":0}}]},` + remote
	g := fixture(t, `{}`, spaces, "", once(`{"action":"gather","category":"food"}`), emptyModel)
	g.Run(t, 3)
	result := g.Risq.Results()
	if result.TurnNumber != g.Risq.TurnNumber() || len(result.Players) != 2 {
		t.Fatalf("results=%+v", result)
	}
	seen := make(map[int]bool)
	for _, player := range result.Players {
		own := g.Own(t, player.PlayerId)
		if seen[player.PlayerId] || player.Nickname != g.Player(t, player.PlayerId).GetNickname() {
			t.Fatalf("identity=%+v", player)
		}
		seen[player.PlayerId] = true
		if player.Units != len(own.Units) || player.Buildings != len(own.Buildings) || player.Eliminated != own.Eliminated {
			t.Fatalf("counts=%+v own=%+v", player, own)
		}
		land, techs := 0, 0
		for _, row := range aisim.Decode(t, aisim.Payload(t, g.Player(t, player.PlayerId))).Spaces {
			for _, space := range row {
				if space != nil && space.Ownership != nil && *space.Ownership == player.PlayerId {
					land++
				}
			}
		}
		for _, researched := range own.ResearchedTechs {
			if researched {
				techs++
			}
		}
		if player.Land != land || player.TechsResearched != techs {
			t.Fatalf("land/techs=%+v", player)
		}
		if player.Gathered.Food != own.Resources.Food || player.Gathered.Wood != 0 || player.Gathered.Stone != 0 || player.Gathered.Gold != 0 {
			t.Fatalf("gathered=%+v stockpile=%+v", player.Gathered, own.Resources)
		}
		for _, value := range []float64{player.Economy.VillagerIdlePct, player.Economy.VillagerIdlePctEarly, player.Economy.VillageCenterIdlePct, player.Economy.VillageCenterIdlePctEarly} {
			if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 100 {
				t.Fatalf("idle percentage=%v", value)
			}
		}
	}
}
