package bridge

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

const ownScout = `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":21,"player":0,"count":1}]}]}`

func snapshotUnitCount(snapshot harness.State, space *harness.SpaceState) (int, bool) {
	if space.UnitCount != nil {
		return *space.UnitCount, true
	}
	if space.Visibility < 3 {
		return 0, false
	}
	count := 0
	for _, player := range snapshot.Players {
		for _, unit := range player.Units {
			if unit.GarrisonedIn == nil && unit.Space == space.Coordinate {
				count++
			}
		}
	}
	return count, true
}

func assertSnapshotCounts(t *testing.T, view ai.View, snapshot harness.State) {
	t.Helper()
	for _, row := range snapshot.Spaces {
		for _, space := range row {
			if space != nil {
				assertSnapshotUnitCount(t, view, snapshot, space)
			}
		}
	}
}

func assertSnapshotUnitCount(t *testing.T, view ai.View, snapshot harness.State, space *harness.SpaceState) {
	t.Helper()
	count, known := snapshotUnitCount(snapshot, space)
	condition := ai.SpaceCondition{UnitCountAtLeast: &count, UnitCountAtMost: &count}
	got := 0
	for _, coordinate := range view.MatchingSpaces(condition) {
		if coordinate.X == space.Coordinate.X && coordinate.Y == space.Coordinate.Y {
			got++
		}
	}
	want := 0
	if known {
		want = 1
	}
	if got != want {
		t.Errorf("space %+v: count query = %d, want %d for (%d, %t)", space.Coordinate, got, want, count, known)
	}
}

func snapshotNeighbors(snapshot harness.State, from ai.Coordinate) []ai.Coordinate {
	seen := map[ai.Coordinate]bool{}
	for _, d := range []ai.Coordinate{{X: 1}, {Y: 1}, {X: -1, Y: 1}, {X: -1}, {Y: -1}, {X: 1, Y: -1}} {
		to := ai.Coordinate{X: from.X + d.X, Y: from.Y + d.Y}
		if snapshot.Space(to.X, to.Y) != nil {
			seen[to] = true
		}
	}
	for _, link := range snapshot.SpaceLinks {
		a, b := ai.Coordinate{X: link.From.X, Y: link.From.Y}, ai.Coordinate{X: link.To.X, Y: link.To.Y}
		if a == from {
			seen[b] = true
		}
		if b == from {
			seen[a] = true
		}
	}
	neighbors := make([]ai.Coordinate, 0, len(seen))
	for to := range seen {
		neighbors = append(neighbors, to)
	}
	sort.Slice(neighbors, func(i, j int) bool {
		return neighbors[i].X < neighbors[j].X || neighbors[i].X == neighbors[j].X && neighbors[i].Y < neighbors[j].Y
	})
	return neighbors
}

func assertSnapshotNeighbors(t *testing.T, view ai.View, snapshot harness.State) {
	t.Helper()
	for _, from := range []ai.Coordinate{{}, {X: 1}, {X: 3}, {X: 99}} {
		want := snapshotNeighbors(snapshot, from)
		got := view.SpaceNeighbors(from)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("neighbors of %+v = %+v, want %+v", from, got, want)
		}
		if len(got) > 0 {
			got[0] = ai.Coordinate{X: 999}
		}
		if next := view.SpaceNeighbors(from); !reflect.DeepEqual(next, want) {
			t.Errorf("neighbor cache mutated: %+v", next)
		}
	}
}

func TestSnapshotCachesMatchRawSnapshotAcrossViewsAndOrders(t *testing.T) {
	g, player := bridgeGame(t, ownScout+`,{"x":1,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":202,"player":1}}]},{"x":3,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":11,"player":1,"count":1}]}]}`)
	for phase := 0; phase < 4; phase++ {
		data, err := json.Marshal(g.Risq.ToFrontend(uint64(g.Human(0)+1), false))
		if err != nil {
			t.Fatal(err)
		}
		var snapshot harness.State
		if err := json.Unmarshal(data, &snapshot); err != nil {
			t.Fatal(err)
		}
		owner, garrison := phase-1, uint64(777)
		snapshot.Space(0, 0).Ownership = &owner
		if phase%2 == 1 {
			snapshot.Player(0).Units[0].GarrisonedIn = &garrison
		}
		snapshot.Space(1, 0).Visibility = phase
		if phase == 2 {
			snapshot.Space(1, 0).UnitCount = &phase
		} else {
			snapshot.Space(1, 0).UnitCount = nil
		}
		if phase == 3 {
			snapshot.Space(1, 0).Zones[0][0].Building = nil
		}
		snapshot.SpaceLinks = []harness.SpaceLink{{From: harness.Coord{}, To: harness.Coord{X: 3}}, {From: harness.Coord{}, To: harness.Coord{X: 3}}, {From: harness.Coord{}, To: harness.Coord{X: 1}}}
		checkCacheSnapshot(t, player, snapshot)
	}
}

func checkCacheSnapshot(t *testing.T, player *game.Player, snapshot harness.State) {
	t.Helper()
	data, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	var payload gin.H
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatal(err)
	}
	probe := &viewProbe{Model: ai.NoopModel{}, Run: func(view ai.View) {
		assertSpaceQueryCaching(t, view)
		for i := 0; i < 3; i++ {
			assertSnapshotCounts(t, view, snapshot)
			assertSnapshotNeighbors(t, view, snapshot)
			view.MoveSpaceOrder(view.Units()[0], ai.Coordinate{X: 1}, true)
		}
	}}
	w := runWorker(t, player, probe, make(chan game.PlayerAction, 4))
	player.AiUpdates <- &game.UpdateMessage{Kind: "start-turn", Content: gin.H{"game": payload}}
	receive(t, w.Actions)
	w.close(t)
}

func assertSpaceQueryCaching(t *testing.T, view ai.View) {
	t.Helper()
	all := len(view.AllSpaces())
	emptyAny := ai.SpaceCondition{Any: []ai.SpaceCondition{}}
	cases := []struct {
		condition ai.SpaceCondition
		count     int
	}{
		{ai.SpaceCondition{}, all}, {emptyAny, 0},
		{ai.SpaceCondition{Not: &emptyAny}, all},
		{ai.SpaceCondition{All: []ai.SpaceCondition{emptyAny}}, 0},
		{ai.SpaceCondition{Any: []ai.SpaceCondition{{}}}, all},
	}
	for repeat := 0; repeat < 2; repeat++ {
		for _, c := range cases {
			if got := len(view.MatchingSpaces(c.condition)); got != c.count {
				t.Errorf("space query %+v = %d, want %d", c.condition, got, c.count)
			}
		}
	}
	vision := uint8(0)
	condition := ai.SpaceCondition{VisionAtLeast: &vision}
	if len(view.MatchingSpaces(condition)) != all {
		t.Error("vision zero should match every space")
	}
	vision = 99
	if len(view.MatchingSpaces(condition)) != 0 {
		t.Error("changed pointed-to condition reused stale matches")
	}
	assertCollectionCopy(t, func() []ai.Coordinate { return view.MatchingSpaces(ai.SpaceCondition{}) }, func(coordinate *ai.Coordinate) { coordinate.X = 999 })
}

func TestSpaceTerrainFilters(t *testing.T) {
	for _, c := range []struct {
		where string
		count int
	}{
		{`{"terrain_types":["flatlands"]}`, 1},
		{`{"terrain_types":["hilly","mountainous"]}`, 1},
		{`{"passable":true}`, 2}, {`{"passable":false}`, 1},
		{`{"intra_cost_at_least":2,"intra_cost_at_most":2}`, 1},
		{`{"inter_cost_at_least":9,"inter_cost_at_most":9}`, 1},
		{`{"build_speed_modifier_at_least":0.8,"build_speed_modifier_at_most":0.8}`, 1},
		{`{"terrain_types":["hilly"],"inter_cost_at_most":6}`, 0},
		{`{"any":[{"terrain_types":["flatlands"]},{"passable":false}]}`, 2},
	} {
		t.Run(c.where, func(t *testing.T) {
			g, player := bridgeGame(t, ownScout+`,{"x":1,"y":0,"terrain":51},{"x":0,"y":1,"terrain":251},{"x":3,"y":0,"terrain":51,"zones":[{"x":0,"y":0,"units":[{"id":11,"player":1,"count":1}]}]}`)
			action := fmt.Sprintf(`{"action":"if","when":{"value_equals":{"value":{"count":{"kind":"spaces","where":%s}},"amount":%d}},"then":[{"action":"select","actors":{"kind":"units"},"issue":[{"delete":{}}]}]}`, c.where, c.count)
			w := runWorker(t, player, parsedModel(t, action), make(chan game.PlayerAction, 1))
			player.AiUpdates <- &game.UpdateMessage{Kind: "start-turn", Content: gin.H{"game": g.Risq.ToFrontend(uint64(g.Human(0)+1), false)}}
			orders := receive(t, w.Actions).Action["orders"].([]defs.OrderFromFrontend)
			if len(orders) != 1 {
				t.Fatalf("terrain count did not equal %d: %+v", c.count, orders)
			}
		})
	}
}

func TestSpaceTerrainFieldsInExpressions(t *testing.T) {
	g, player := bridgeGame(t, ownScout+`,{"x":1,"y":0,"terrain":51},{"x":0,"y":1,"terrain":251},{"x":3,"y":0,"terrain":51,"zones":[{"x":0,"y":0,"units":[{"id":11,"player":1,"count":1}]}]}`)
	score := "var(s.terrain_type)+var(s.intra_cost)+var(s.inter_cost)+var(s.build_speed_modifier)+var(s.passable)"
	action := fmt.Sprintf(`{"action":"for_each","source":"spaces","as":"s","where":{"terrain_types":["hilly"]},"do":[{"action":"if","when":{"value_at_least":{"value":%q,"amount":14.7}},"then":[{"action":"select","actors":{"kind":"units"},"issue":[{"delete":{}}]}]}]}`, score)
	seen := &recordingModel{Model: parsedModel(t, action), Seen: make(chan observation, 1)}
	w := runWorker(t, player, seen, make(chan game.PlayerAction, 1))
	player.AiUpdates <- &game.UpdateMessage{Kind: "start-turn", Content: gin.H{"game": g.Risq.ToFrontend(uint64(g.Human(0)+1), false)}}
	observed := receive(t, seen.Seen)
	orders := receive(t, w.Actions).Action["orders"].([]defs.OrderFromFrontend)
	if len(orders) != 1 {
		t.Fatalf("terrain fields were unavailable in loop or target expressions: %+v", orders)
	}
	for _, info := range observed.Spaces {
		if info.Space == (ai.Coordinate{X: 1}) && (info.TerrainType != uint8(defs.TerrainType_HILLY) || info.IntraCost != 2 || info.InterCost != 9 || info.BuildSpeedModifier != 0.8 || !info.Passable) {
			t.Fatalf("incorrect terrain fields: %+v", info)
		}
	}
}

func TestRealPayloadVisibility(t *testing.T) {
	for _, c := range []struct {
		name   string
		x      int
		vision uint8
	}{{"unexplored", 3, 0}, {"count", 1, 2}, {"full", 0, 3}, {"spy", 0, 4}} {
		t.Run(c.name, func(t *testing.T) {
			own := ""
			if c.x == 0 {
				own = `,{"id":21,"player":0,"count":1}`
			}
			spaces := fmt.Sprintf(`{"x":%d,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"resource":1,"units":[{"id":11,"player":1,"count":1}%s]},{"x":1,"y":0,"building":{"id":202,"player":1}}]}`, c.x, own)
			if c.x != 0 {
				spaces += "," + ownScout
			}
			g, player := bridgeGame(t, spaces, func() {
				if c.vision == 4 {
					config := defs.UnitConfigs[21]
					config.Vision.Space = defs.VisibilitySpy
					defs.UnitConfigs[21] = config
				}
			})
			g.Action(g.Human(0), "set-unit-behavior", gin.H{"internal_ids": []uint64{g.Self(g.Human(0)).Units[0].InternalID}, "stance": uint8(defs.UnitStance_PASSIVE), "attack_back": false})
			p := g.Human(1)
			g.Submit(p, harness.OrderGather([]uint64{g.Self(p).Units[0].InternalID}, c.x, 0, 0, 0))
			g.EndTurn()
			payload := g.Risq.ToFrontend(uint64(g.Human(0)+1), false)
			seen := &recordingModel{Model: parsedModel(t, `{"action":"select","actors":{"kind":"units","limit":1},"pick":{"kind":"units","where":{"owner":"enemy"}},"issue":[{"attack":{}}]}`), Seen: make(chan observation, 1)}
			w := runWorker(t, player, seen, make(chan game.PlayerAction, 4))
			player.AiUpdates <- &game.UpdateMessage{Kind: "start-turn", Content: gin.H{"game": payload}}
			observed := receive(t, seen.Seen)
			action := receive(t, w.Actions)
			want := 0
			if c.vision >= 3 {
				want = 1
			}
			if orders := action.Action["orders"].([]defs.OrderFromFrontend); len(orders) != want {
				t.Fatalf("vision %d orders=%+v", c.vision, orders)
			}
			assertVisibility(t, observed, c.x, c.vision)
		})
	}
}

func TestBridgeKeepsLastKnownBuildingsAfterCoverageIsLost(t *testing.T) {
	spaces := `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":11,"player":0,"count":1}]},{"x":1,"y":0,"building":{"id":202,"player":1}}]},{"x":-3,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":11,"player":0,"count":1}]}]},{"x":3,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":11,"player":1,"count":1}]}]}`
	g, player := bridgeGame(t, spaces)
	seen := &recordingModel{Model: ai.NoopModel{}, Seen: make(chan observation, 1)}
	w := runWorker(t, player, seen, make(chan game.PlayerAction, 4))
	owner, enemy := g.Human(0), g.Human(1)
	player.AiUpdates <- &game.UpdateMessage{Kind: "start-turn", Content: gin.H{"game": g.Risq.ToFrontend(uint64(owner+1), false)}}
	initial := receive(t, seen.Seen)
	receive(t, w.Actions)
	if len(initial.VisibleBuildings) != 1 {
		t.Fatalf("initial buildings=%+v", initial)
	}
	var scout uint64
	for _, unit := range g.Self(owner).Units {
		if unit.Space.X == 0 {
			scout = unit.InternalID
		}
	}
	if scout == 0 {
		t.Fatal("missing scout")
	}
	g.Submit(owner, harness.Order(defs.OrderType_UnitDelete, []uint64{scout}, 0, false))
	g.EndTurn()
	g.Submit(enemy, harness.Order(defs.OrderType_BuildingDelete, []uint64{g.Self(enemy).Buildings[0].InternalID}, 0, false))
	g.EndTurn()
	g.EndTurn()
	if g.State(owner).Space(0, 0).Visibility != 1 {
		t.Fatal("fixture never lost coverage")
	}
	player.AiUpdates <- &game.UpdateMessage{Kind: "start-turn", Content: gin.H{"game": g.Risq.ToFrontend(uint64(owner+1), false)}}
	fog := receive(t, seen.Seen)
	receive(t, w.Actions)
	if len(fog.VisibleBuildings) != 0 || len(fog.Units) != 0 || len(fog.KnownBuildings) != 1 || fog.KnownBuildings[0].InternalID != initial.KnownBuildings[0].InternalID {
		t.Fatalf("fog=%+v", fog)
	}
}

func assertVisibility(t *testing.T, seen observation, x int, vision uint8) {
	t.Helper()
	found := false
	for _, space := range seen.Spaces {
		if space.Space.X == x && space.Space.Y == 0 {
			found = true
			if space.Vision != vision {
				t.Fatalf("vision=%d, want %d", space.Vision, vision)
			}
			if vision == 2 && (space.UnitCount == nil || *space.UnitCount != 1) {
				t.Fatalf("count=%+v", space)
			}
		}
	}
	if !found {
		t.Fatal("target space missing")
	}
	if vision == 0 {
		if len(seen.Units)+len(seen.VisibleBuildings)+len(seen.KnownBuildings)+len(seen.Resources) != 0 {
			t.Fatalf("unexplored leaked: %+v", seen)
		}
		return
	}
	if len(seen.Resources) != 1 || len(seen.VisibleBuildings) != 1 || len(seen.KnownBuildings) != 1 {
		t.Fatalf("known contents=%+v", seen)
	}
	if vision == 2 {
		if len(seen.Units) != 0 {
			t.Fatalf("count vision exposed units: %+v", seen.Units)
		}
		return
	}
	if len(seen.Units) != 1 {
		t.Fatalf("visible units=%+v", seen.Units)
	}
	if (seen.Units[0].CurrentOrder != nil) != (vision == 4) {
		t.Fatalf("enemy order at vision %d: %+v", vision, seen.Units[0])
	}
}

func TestUnseenChangesDoNotChangeViewOrDecision(t *testing.T) {
	var observations []observation
	for count := 1; count <= 2; count++ {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			spaces := ownScout + fmt.Sprintf(`,{"x":3,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"resource":%d,"units":[{"id":11,"player":1,"count":%d}]},{"x":1,"y":0,"building":{"id":202,"player":1}}]}`, count, count)
			g, player := bridgeGame(t, spaces)
			seen := &recordingModel{Model: parsedModel(t, `{"action":"select","actors":{"kind":"units"},"pick":{"kind":"units","where":{"owner":"enemy"}},"issue":[{"attack":{}}]}`), Seen: make(chan observation, 1)}
			w := runWorker(t, player, seen, make(chan game.PlayerAction, 4))
			player.AiUpdates <- &game.UpdateMessage{Kind: "start-turn", Content: gin.H{"game": g.Risq.ToFrontend(uint64(g.Human(0)+1), false)}}
			view := receive(t, seen.Seen)
			orders := receive(t, w.Actions).Action["orders"].([]defs.OrderFromFrontend)
			if len(orders) != 0 {
				t.Fatalf("hidden target orders=%+v", orders)
			}
			sort.Slice(view.Spaces, func(i, j int) bool {
				a, b := view.Spaces[i].Space, view.Spaces[j].Space
				return a.X < b.X || a.X == b.X && a.Y < b.Y
			})
			observations = append(observations, view)
		})
	}
	if len(observations) != 2 || !reflect.DeepEqual(observations[0], observations[1]) {
		t.Fatalf("hidden changes leaked: %+v", observations)
	}
}
