package bridge

import (
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

const ownScout = `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":101,"player":0,"count":1}]}]}`

func TestRealPayloadVisibility(t *testing.T) {
	for _, c := range []struct {
		name   string
		x      int
		vision uint8
	}{{"unexplored", 3, 0}, {"count", 1, 2}, {"full", 0, 3}, {"spy", 0, 4}} {
		t.Run(c.name, func(t *testing.T) {
			own := ""
			if c.x == 0 {
				own = `,{"id":101,"player":0,"count":1}`
			}
			spaces := fmt.Sprintf(`{"x":%d,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"resource":1,"units":[{"id":1,"player":1,"count":1}%s]},{"x":1,"y":0,"building":{"id":2,"player":1}}]}`, c.x, own)
			if c.x != 0 {
				spaces += "," + ownScout
			}
			g, player := bridgeGame(t, spaces, func() {
				if c.vision == 4 {
					config := defs.UnitConfigs[101]
					config.Vision.Space = defs.VisibilitySpy
					defs.UnitConfigs[101] = config
				}
			})
			g.Action(g.Human(0), "set-unit-behavior", gin.H{"internal_ids": []uint64{g.Self(g.Human(0)).Units[0].InternalID}, "stance": uint8(defs.UnitStance_PASSIVE), "attack_back": false})
			p := g.Human(1)
			g.Submit(p, harness.OrderGather([]uint64{g.Self(p).Units[0].InternalID}, c.x, 0, 0, 0))
			g.EndTurn()
			payload := g.Risq.ToFrontend(uint64(g.Human(0)+1), false)
			seen := &recordingModel{Model: parsedModel(t, `{"action":"attack","targets":"enemy_units","score":1,"max":1}`), Seen: make(chan observation, 1)}
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
	spaces := `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":1,"player":0,"count":1}]},{"x":1,"y":0,"building":{"id":2,"player":1}}]},{"x":-3,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":1,"player":0,"count":1}]}]},{"x":3,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":1,"player":1,"count":1}]}]}`
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
			spaces := ownScout + fmt.Sprintf(`,{"x":3,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"resource":%d,"units":[{"id":1,"player":1,"count":%d}]},{"x":1,"y":0,"building":{"id":2,"player":1}}]}`, count, count)
			g, player := bridgeGame(t, spaces)
			seen := &recordingModel{Model: parsedModel(t, `{"action":"attack","targets":"enemy_units","score":1}`), Seen: make(chan observation, 1)}
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
