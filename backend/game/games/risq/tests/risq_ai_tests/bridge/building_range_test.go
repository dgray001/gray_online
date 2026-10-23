package bridge

import (
	"fmt"
	"slices"
	"testing"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"github.com/gin-gonic/gin"
)

const watchtower = 207

type rangeProbe struct {
	ai.Model
	Targets []ai.ZoneRef
	Results chan []bool
}

func (m *rangeProbe) DecideOrders(view ai.View) ai.Decision {
	row := make([]bool, len(m.Targets))
	for i, target := range m.Targets {
		row[i] = view.InAttackRange(view.Buildings()[0], target)
	}
	m.Results <- row
	return m.Model.DecideOrders(view)
}

func zoneRef(spaceX, spaceY, zoneX, zoneY int) ai.ZoneRef {
	return ai.ZoneRef{Space: ai.Coordinate{X: spaceX, Y: spaceY}, Zone: ai.Coordinate{X: zoneX, Y: zoneY}}
}

func TestInAttackRangeForSpaceRangeBuilding(t *testing.T) {
	targets := []ai.ZoneRef{zoneRef(0, 0, -1, 0), zoneRef(1, 0, -1, 0), zoneRef(1, 0, 0, 0), zoneRef(1, 0, 0, 1), zoneRef(-1, 0, 1, 0)}
	for _, c := range []struct {
		name   string
		tower  [2]int
		inside []bool
	}{
		{"edge tower reaches own space and the facing zone", [2]int{1, 0}, []bool{true, true, false, false, false}},
		{"center tower reaches only its own space", [2]int{0, 0}, []bool{true, false, false, false, false}},
	} {
		t.Run(c.name, func(t *testing.T) {
			spaces := fmt.Sprintf(`{"x":0,"y":0,"terrain":1,"zones":[{"x":%d,"y":%d,"building":{"id":%d,"player":0}},{"x":-1,"y":1,"units":[{"id":11,"player":0,"count":1}]}]},{"x":1,"y":0,"terrain":1,"zones":[{"x":-1,"y":0,"units":[{"id":22,"player":1,"count":1}]}]}`, c.tower[0], c.tower[1], watchtower)
			g, player := bridgeGame(t, spaces)
			probe := &rangeProbe{Model: ai.NoopModel{}, Targets: targets, Results: make(chan []bool, 1)}
			w := runWorker(t, player, probe, make(chan game.PlayerAction, 4))
			player.AiUpdates <- &game.UpdateMessage{Kind: "start-turn", Content: gin.H{"game": g.Risq.ToFrontend(uint64(g.Human(0)+1), false)}}
			got := receive(t, probe.Results)
			receive(t, w.Actions)
			for i, want := range c.inside {
				if got[i] != want {
					t.Errorf("target %+v: in range=%v, want %v", targets[i], got[i], want)
				}
			}
		})
	}
}

type buildZoneProbe struct {
	ai.Model
	Results chan []bool
}

func (m *buildZoneProbe) DecideOrders(view ai.View) ai.Decision {
	zone := func(x int, y int) ai.ZoneRef { return ai.ZoneRef{Zone: ai.Coordinate{X: x, Y: y}} }
	m.Results <- []bool{
		view.ValidBuildZone(211, zone(0, 0)), view.ValidBuildZone(211, zone(0, 1)), view.ValidBuildZone(202, zone(0, 0)),
		view.ValidBuildZone(202, zone(1, -1)), view.ValidBuildZone(202, zone(-1, 0)), view.ValidBuildZone(202, zone(5, 5)),
	}
	return m.Model.DecideOrders(view)
}

type sitesProbe struct {
	ai.Model
	Seen chan [2][]ai.ZoneRef
}

func (m *sitesProbe) DecideOrders(view ai.View) ai.Decision {
	m.Seen <- [2][]ai.ZoneRef{view.BuildSites(211, nil), view.BuildSites(202, nil)}
	return m.Model.DecideOrders(view)
}

type factsProbe struct {
	ai.Model
	Seen chan factsSeen
}

type factsSeen struct {
	tower                    ai.BuildingView
	own, enemy               ai.UnitView
	all_units, all_buildings int
}

func (m *factsProbe) DecideOrders(view ai.View) ai.Decision {
	facts := factsSeen{tower: view.Buildings()[0], all_units: len(view.AllUnits()), all_buildings: len(view.AllBuildings())}
	for _, u := range view.AllUnits() {
		if u.PlayerID == view.PlayerID() {
			facts.own = u
		} else {
			facts.enemy = u
		}
	}
	m.Seen <- facts
	return m.Model.DecideOrders(view)
}

func TestValidBuildZoneMirrorsValidation(t *testing.T) {
	home := `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":21,"player":0,"count":1}]},{"x":1,"y":-1,"resource":11},{"x":-1,"y":0,"building":{"id":203,"player":1}}]}`
	g, player := bridgeGame(t, home+`,{"x":3,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":11,"player":1,"count":1}]}]}`)
	probe := &buildZoneProbe{Model: ai.NoopModel{}, Results: make(chan []bool, 1)}
	w := runWorker(t, player, probe, make(chan game.PlayerAction, 4))
	player.AiUpdates <- &game.UpdateMessage{Kind: "start-turn", Content: gin.H{"game": g.Risq.ToFrontend(uint64(g.Human(0)+1), false)}}
	got := receive(t, probe.Results)
	receive(t, w.Actions)
	for i, want := range []bool{false, true, true, false, false, false} {
		if got[i] != want {
			t.Errorf("case %d: valid=%v, want %v (wall center, wall edge, house center, known resource, known building, missing zone)", i, got[i], want)
		}
	}
}

func TestWallsAreNeverOfferedACenterZone(t *testing.T) {
	g, player := bridgeGame(t, ownScout+`,{"x":3,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":11,"player":1,"count":1}]}]}`)
	probe := &sitesProbe{Model: ai.NoopModel{}, Seen: make(chan [2][]ai.ZoneRef, 1)}
	w := runWorker(t, player, probe, make(chan game.PlayerAction, 4))
	player.AiUpdates <- &game.UpdateMessage{Kind: "start-turn", Content: gin.H{"game": g.Risq.ToFrontend(uint64(g.Human(0)+1), false)}}
	sites := receive(t, probe.Seen)
	receive(t, w.Actions)
	hasCenter := func(zones []ai.ZoneRef) bool {
		return slices.ContainsFunc(zones, func(z ai.ZoneRef) bool { return z.Zone == (ai.Coordinate{}) })
	}
	if len(sites[0]) == 0 || hasCenter(sites[0]) {
		t.Errorf("wall sites %v, want edge zones only", sites[0])
	}
	if !hasCenter(sites[1]) {
		t.Errorf("ordinary building sites %v, want center zones included", sites[1])
	}
}

func TestViewExposesRangeStaminaAndUnifiedLists(t *testing.T) {
	spaces := fmt.Sprintf(`{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":%d,"player":0}},{"x":-1,"y":1,"units":[{"id":11,"player":0,"count":1}]}]},{"x":1,"y":0,"terrain":1,"zones":[{"x":-1,"y":0,"units":[{"id":22,"player":1,"count":1}]}]}`, watchtower)
	g, player := bridgeGame(t, spaces)
	probe := &factsProbe{Model: ai.NoopModel{}, Seen: make(chan factsSeen, 1)}
	w := runWorker(t, player, probe, make(chan game.PlayerAction, 4))
	player.AiUpdates <- &game.UpdateMessage{Kind: "start-turn", Content: gin.H{"game": g.Risq.ToFrontend(uint64(g.Human(0)+1), false)}}
	facts := receive(t, probe.Seen)
	receive(t, w.Actions)
	if facts.tower.AttackRange != ai.AttackRangeSpace || !facts.tower.Visible {
		t.Errorf("own watchtower range %v visible %v; want space range and visible", facts.tower.AttackRange, facts.tower.Visible)
	}
	if facts.own.MaxStamina <= 0 {
		t.Errorf("own unit max stamina %v, want positive", facts.own.MaxStamina)
	}
	if facts.enemy.InternalID == 0 || facts.enemy.MaxStamina != 0 {
		t.Errorf("enemy unit %+v, want it listed with its stamina hidden", facts.enemy)
	}
	if facts.all_units != 2 || facts.all_buildings != 1 {
		t.Errorf("AllUnits %d AllBuildings %d, want 2 and 1", facts.all_units, facts.all_buildings)
	}
}
