package economy

import (
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

const townHallId = 201

// Player 0 holds space (0,0) with workers and a military unit; extra_spaces are appended after player 1's lone worker
func limitGame(t *testing.T, workers int, extra_spaces string) *harness.Game {
	t.Helper()
	doc := fmt.Sprintf(`{"board_size":3,"players":2,"starting_bank":{"wood":1000,"stone":1000},"spaces":[{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":11,"player":0,"count":%d},{"id":21,"player":0,"count":1}]}]},{"x":-3,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":11,"player":1,"count":1}]}]}%s]}`, workers, extra_spaces)
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"limit": doc})
	return startGame(t, "custom:limit", 1, 2)
}

func workerIds(g *harness.Game, human int) []uint64 {
	var ids []uint64
	for _, u := range g.Self(human).Units {
		if u.UnitID == 11 {
			ids = append(ids, u.InternalID)
		}
	}
	return ids
}

func townHalls(g *harness.Game, human int) []harness.Building {
	var centers []harness.Building
	for _, b := range g.Self(human).Buildings {
		if b.BuildingID == townHallId {
			centers = append(centers, b)
		}
	}
	return centers
}

func TestWorkerBuildsTownHallWhenNoneOwned(t *testing.T) {
	g := limitGame(t, 1, "")
	p := g.Human(0)
	g.Submit(p, harness.OrderBuild(workerIds(g, p), townHallId, 0, 0, 0, 0))
	g.EndTurn()
	if centers := townHalls(g, p); len(centers) != 1 || !centers[0].UnderConstruction {
		t.Fatalf("town halls %+v, want one under construction", centers)
	}
	if wood, want := g.Self(p).Resources.Wood, 1000-defs.BuildingConfigs[townHallId].Cost.Wood; wood != want {
		t.Errorf("wood %v, want %v", wood, want)
	}
}

func TestSecondTownHallRejectedAtReceiptWhenOneOwned(t *testing.T) {
	g := limitGame(t, 1, `,{"x":1,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":201,"player":0}}]}`)
	p := g.Human(0)
	g.Submit(p, harness.OrderBuild(workerIds(g, p), townHallId, 0, 0, 0, 0))
	g.EndTurn()
	state := g.Self(p)
	if centers := townHalls(g, p); len(centers) != 1 {
		t.Errorf("town halls %+v, want only the original", centers)
	}
	if state.Resources.Wood != 1000 || len(state.PlannedFoundations) != 0 {
		t.Errorf("wood %v, foundations %v; want no charge and no reservation", state.Resources.Wood, state.PlannedFoundations)
	}
}

func TestSecondTownHallRejectedAtReceiptInSameBatch(t *testing.T) {
	g := limitGame(t, 2, "")
	p := g.Human(0)
	ids := workerIds(g, p)
	g.Submit(p,
		harness.OrderBuild(ids[:1], townHallId, 0, 0, 0, 0),
		harness.OrderBuild(ids[1:], townHallId, 0, 0, 1, 0),
	)
	g.EndTurn()
	cost := defs.BuildingConfigs[townHallId].Cost
	if centers := townHalls(g, p); len(centers) != 1 {
		t.Errorf("town halls %+v, want exactly one", centers)
	}
	if state := g.Self(p); state.Resources.Wood != 1000-cost.Wood || len(state.PlannedFoundations) != 0 {
		t.Errorf("wood %v, foundations %v; want one charge and no reservation left", state.Resources.Wood, state.PlannedFoundations)
	}
}

func TestWorkersMayAssistOwnTownHallFoundation(t *testing.T) {
	g := limitGame(t, 2, "")
	p := g.Human(0)
	g.Submit(p, harness.OrderBuild(workerIds(g, p), townHallId, 0, 0, 0, 0))
	g.EndTurn()
	if centers := townHalls(g, p); len(centers) != 1 {
		t.Fatalf("town halls %+v, want one shared foundation", centers)
	}
	if wood, want := g.Self(p).Resources.Wood, 1000-defs.BuildingConfigs[townHallId].Cost.Wood; wood != want {
		t.Errorf("wood %v, want a single charge %v", wood, want)
	}
}

func TestTownHallBuildableAgainAfterLoss(t *testing.T) {
	g := limitGame(t, 1, `,{"x":1,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":201,"player":0}}]}`)
	p := g.Human(0)
	existing := townHalls(g, p)[0]
	g.Submit(p, harness.Order(defs.OrderType_BuildingDelete, []uint64{existing.InternalID}, 0, false))
	g.EndTurn()
	if centers := townHalls(g, p); len(centers) != 0 {
		t.Fatalf("town halls %+v after delete, want none", centers)
	}
	g.Submit(p, harness.OrderBuild(workerIds(g, p), townHallId, 0, 0, 0, 0))
	g.EndTurn()
	if centers := townHalls(g, p); len(centers) != 1 {
		t.Errorf("town halls %+v, want a rebuilt one", centers)
	}
}
