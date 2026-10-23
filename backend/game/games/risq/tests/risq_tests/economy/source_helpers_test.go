package economy

import (
	"fmt"
	"maps"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

const fastGatherSpeed = 250

// Pools keep their size but empty in a couple of turns
func speedUpGathering(t *testing.T) {
	origRes, origOrchard := maps.Clone(defs.ResourceConfigs), defs.BuildingConfigs[203]
	t.Cleanup(func() { defs.ResourceConfigs, defs.BuildingConfigs[203] = origRes, origOrchard })
	for id, config := range defs.ResourceConfigs {
		config.Base_gather_speed = fastGatherSpeed
		defs.ResourceConfigs[id] = config
	}
	orchard := defs.BuildingConfigs[203]
	orchard.Gather.Base_gather_speed = fastGatherSpeed
	defs.BuildingConfigs[203] = orchard
}

func sourceGame(t *testing.T, orchard bool) *harness.Game {
	t.Helper()
	source, workers := `"resource":11`, 3
	if orchard {
		source, workers = `"building":{"id":203,"player":0}`, 1
	}
	doc := fmt.Sprintf(`{"board_size":2,"players":2,"starting_bank":{"wood":120},"spaces":[{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,%s,"units":[{"id":11,"player":0,"count":%d}]}]},{"x":2,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":11,"player":1,"count":1}]}]}]}`, source, workers)
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"source": doc})
	return startGame(t, "custom:source", 1, 2)
}

func orchardZone(g *harness.Game, human int) harness.ZoneState {
	g.T.Helper()
	for _, row := range g.State(human).Space(0, 0).Zones {
		for _, zone := range row {
			if zone.Coordinate == (harness.Coord{}) && zone.Building != nil {
				return zone
			}
		}
	}
	g.T.Fatal("orchard missing from center zone")
	return harness.ZoneState{}
}

func exhaustOrchard(g *harness.Game, human int) uint64 {
	g.T.Helper()
	speedUpGathering(g.T)
	id := orchardZone(g, human).Building.InternalID
	submitGather(g, human)
	for turn := 0; turn < 60 && orchardZone(g, human).Building.ResourcesLeft > 0; turn++ {
		g.EndTurn()
	}
	if orchardZone(g, human).Building.ResourcesLeft != 0 {
		g.T.Fatal("orchard not depleted within 60 turns")
	}
	return id
}

func fastExhaustOrchard(g *harness.Game, human int) uint64 {
	g.T.Helper()
	original := defs.BuildingConfigs[203]
	config := original
	config.Gather.Base_gather_speed = 250
	defs.BuildingConfigs[203] = config
	defer func() { defs.BuildingConfigs[203] = original }()
	return exhaustOrchard(g, human)
}
