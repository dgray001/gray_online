package economy

import (
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

func sourceGame(t *testing.T, farm bool) *harness.Game {
	t.Helper()
	source, workers := `"resource":11`, 3
	if farm {
		source, workers = `"building":{"id":3,"player":0}`, 1
	}
	doc := fmt.Sprintf(`{"board_size":2,"players":2,"starting_bank":{"wood":120},"spaces":[{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,%s,"units":[{"id":1,"player":0,"count":%d}]}]},{"x":2,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":1,"player":1,"count":1}]}]}]}`, source, workers)
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"source": doc})
	return harness.NewGame(t, "custom:source", 1, 2)
}

func farmZone(g *harness.Game, human int) harness.ZoneState {
	g.T.Helper()
	for _, row := range g.State(human).Space(0, 0).Zones {
		for _, zone := range row {
			if zone.Coordinate == (harness.Coord{}) && zone.Building != nil {
				return zone
			}
		}
	}
	g.T.Fatal("farm missing from center zone")
	return harness.ZoneState{}
}

func exhaustFarm(g *harness.Game, human int) uint64 {
	g.T.Helper()
	id := farmZone(g, human).Building.InternalID
	submitGather(g, human)
	for turn := 0; turn < 60 && farmZone(g, human).Building.ResourcesLeft > 0; turn++ {
		g.EndTurn()
	}
	if farmZone(g, human).Building.ResourcesLeft != 0 {
		g.T.Fatal("farm not depleted within 60 turns")
	}
	return id
}

func fastExhaustFarm(g *harness.Game, human int) uint64 {
	g.T.Helper()
	original := defs.BuildingConfigs[3]
	config := original
	config.Gather.Base_gather_speed = 250
	defs.BuildingConfigs[3] = config
	defer func() { defs.BuildingConfigs[3] = original }()
	return exhaustFarm(g, human)
}
