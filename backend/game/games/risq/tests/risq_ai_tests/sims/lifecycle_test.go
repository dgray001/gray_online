package sims

import "testing"

const duel = `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":13,"player":0,"count":1},{"id":1,"player":1,"count":1}]},{"x":1,"y":0,"building":{"id":2,"player":0}},{"x":-1,"y":0,"building":{"id":2,"player":1}}]}`

func TestGameEndStopsAllAIWorkers(t *testing.T) {
	model := once(`{"action":"set_unit_behavior","stance":"passive","attack_back":false},{"action":"attack","target":"economic","max":1}`)
	g := fixture(t, `{}`, duel, "", model, emptyModel)
	g.Run(t, 1)
	if !g.Base.GameEnded() || !g.Own(t, 1).Eliminated || g.Own(t, 0).Eliminated {
		t.Fatalf("end state=%+v", g.Risq.Results())
	}
	g.Stop(t)
	g.Stop(t)
	select {
	case <-g.Base.GameEndedChannel:
	default:
		t.Fatal("missing game-end notification")
	}
}

func TestEliminatedAIStopsWhileSurvivorsContinue(t *testing.T) {
	model := once(`{"action":"set_unit_behavior","stance":"passive","attack_back":false},{"action":"attack","targets":"enemy_units","score":1,"max":1}`)
	third := `{"x":3,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":1,"player":2},"units":[{"id":1,"player":2,"count":1}]}]}`
	g := fixture(t, `{}`, duel+","+third, "", model, emptyModel, emptyModel)
	g.Run(t, 2)
	if g.Base.GameEnded() || !g.Own(t, 1).Eliminated {
		t.Fatalf("elimination=%+v", g.Risq.Results())
	}
	assertSubmissions(t, g, 1, 1)
	assertSubmissions(t, g, 2, 2)
	g.Stop(t)
}
