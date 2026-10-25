package bridge

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/util"
	"github.com/gin-gonic/gin"
)

type unitsSample struct {
	Before []byte
	Same   []byte
	After  []ai.UnitView
}

type unitsProbe struct {
	ai.Model
	Out chan unitsSample
}

func marshalUnits(units []ai.UnitView) []byte {
	data, _ := json.Marshal(units)
	return data
}

func (m *unitsProbe) DecideOrders(view ai.View) ai.Decision {
	first := view.Units()
	before := marshalUnits(first)
	decision := m.Model.DecideOrders(view)
	m.Out <- unitsSample{Before: before, Same: marshalUnits(first), After: view.Units()}
	return decision
}

type orderFacts struct {
	enemy_unit, enemy_building uint64
	order                      *ai.CurrentOrder
}

type orderProbe struct {
	ai.Model
	Seen chan orderFacts
}

func (m *orderProbe) DecideOrders(view ai.View) ai.Decision {
	facts := orderFacts{order: view.Units()[0].CurrentOrder}
	if units := view.VisibleEnemyUnits(); len(units) > 0 {
		facts.enemy_unit = units[0].InternalID
	}
	if buildings := view.VisibleEnemyBuildings(); len(buildings) > 0 {
		facts.enemy_building = buildings[0].InternalID
	}
	m.Seen <- facts
	return m.Model.DecideOrders(view)
}

type foundationProbe struct {
	ai.Model
	Seen chan []ai.FoundationView
}

func (m *foundationProbe) DecideOrders(view ai.View) ai.Decision {
	view.BuildOrder(view.Units()[0], 2, ai.ZoneRef{Zone: ai.Coordinate{X: 1}}, false)
	m.Seen <- view.Foundations()
	return m.Model.DecideOrders(view)
}

func TestFoundationOrderedThisTurnIsFlagged(t *testing.T) {
	g, player := bridgeGame(t, ownScout+`,{"x":3,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":11,"player":1,"count":1}]}]}`)
	probe := &foundationProbe{Model: ai.NoopModel{}, Seen: make(chan []ai.FoundationView, 1)}
	w := runWorker(t, player, probe, make(chan game.PlayerAction, 4))
	player.AiUpdates <- &game.UpdateMessage{Kind: "start-turn", Content: gin.H{"game": g.Risq.ToFrontend(uint64(g.Human(0)+1), false)}}
	foundations := receive(t, probe.Seen)
	receive(t, w.Actions)
	if len(foundations) != 1 || !foundations[0].Planned || !foundations[0].OrderedThisTurn {
		t.Fatalf("foundations = %+v, want one planned foundation flagged as ordered this turn", foundations)
	}
}

func withUnitOrder(t *testing.T, payload gin.H, order_type defs.OrderType, target_id int64) gin.H {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	var snapshot gin.H
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatal(err)
	}
	for _, p := range snapshot["players"].([]any) {
		player := p.(map[string]any)
		if player["player"].(map[string]any)["player_id"].(float64) != 0 {
			continue
		}
		for _, u := range player["units"].([]any) {
			order := map[string]any{"internal_id": 9999, "order_type": order_type, "target_id": target_id}
			u.(map[string]any)["active_orders"] = []any{order}
		}
	}
	return snapshot
}

func TestUnitsCacheSurvivesDecisionAndInvalidatesOnAssign(t *testing.T) {
	spaces := ownScout + `,{"x":1,"y":0,"terrain":1,"zones":[{"x":0,"y":0}]},{"x":3,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":11,"player":1,"count":1}]}]}`
	g, player := bridgeGame(t, spaces)
	model := parsedModel(t, `{"action":"move","targets":"coordinate","coordinate":{"x":1,"y":0}}`)
	probe := &unitsProbe{Model: model, Out: make(chan unitsSample, 1)}
	w := runWorker(t, player, probe, make(chan game.PlayerAction, 4))
	payload := g.Risq.ToFrontend(uint64(g.Human(0)+1), false)
	player.AiUpdates <- &game.UpdateMessage{Kind: "start-turn", Content: gin.H{"game": payload}}
	sample := receive(t, probe.Out)
	receive(t, w.Actions)
	if string(sample.Before) != string(sample.Same) {
		t.Fatalf("a rule mutated the shared Units() slice:\nbefore=%s\nafter=%s", sample.Before, sample.Same)
	}
	if len(sample.After) == 0 || sample.After[0].CurrentOrder == nil {
		t.Fatalf("Units() was stale after the unit was assigned an order: %+v", sample.After)
	}
}

func TestOrderingOneUnitLeavesOtherUnitViewsAlone(t *testing.T) {
	pair := `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":21,"player":0,"count":2}]}]},{"x":3,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":11,"player":1,"count":1}]}]}`
	g, player := bridgeGame(t, pair)
	var others_before, others_after []byte
	var ordered ai.UnitView
	probe := &viewProbe{Model: ai.NoopModel{}, Run: func(view ai.View) {
		units := view.Units()
		others_before = marshalUnits(units[1:])
		view.MoveSpaceOrder(units[0], spaceA, true)
		units = view.Units()
		others_after, ordered = marshalUnits(units[1:]), units[0]
	}}
	w := runWorker(t, player, probe, make(chan game.PlayerAction, 4))
	player.AiUpdates <- &game.UpdateMessage{Kind: "start-turn", Content: gin.H{"game": g.Risq.ToFrontend(uint64(g.Human(0)+1), false)}}
	receive(t, w.Actions)
	if ordered.CurrentOrder == nil {
		t.Error("the ordered unit's view did not show its new order")
	}
	if string(others_before) != string(others_after) {
		t.Errorf("another unit's view changed:\nbefore=%s\nafter=%s", others_before, others_after)
	}
}

func TestCurrentOrderCarriesTheTargetOfTargetedOrders(t *testing.T) {
	spaces := `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":21,"player":0,"count":1}]},{"x":1,"y":0,"units":[{"id":21,"player":1,"count":1}]},{"x":-1,"y":0,"building":{"id":207,"player":1}}]}`
	g, player := bridgeGame(t, spaces)
	probe := &orderProbe{Model: ai.NoopModel{}, Seen: make(chan orderFacts, 1)}
	w := runWorker(t, player, probe, make(chan game.PlayerAction, 4))
	payload := g.Risq.ToFrontend(uint64(g.Human(0)+1), false)
	decide := func(game_payload gin.H) orderFacts {
		player.AiUpdates <- &game.UpdateMessage{Kind: "start-turn", Content: gin.H{"game": game_payload}}
		facts := receive(t, probe.Seen)
		receive(t, w.Actions)
		return facts
	}
	ids := decide(payload)
	space := int64(util.Pair(2, 3))
	for _, c := range []struct {
		order_type defs.OrderType
		target     int64
		reached    func(*ai.CurrentOrder) bool
	}{
		{defs.OrderType_UnitMoveSpace, space, func(o *ai.CurrentOrder) bool {
			return o.TargetSpace != nil && *o.TargetSpace == ai.Coordinate{X: 2, Y: 3}
		}},
		{defs.OrderType_UnitAttackSpace, space, func(o *ai.CurrentOrder) bool {
			return o.TargetSpace != nil && *o.TargetSpace == ai.Coordinate{X: 2, Y: 3}
		}},
		{defs.OrderType_UnitAutoAttackUnit, int64(ids.enemy_unit), func(o *ai.CurrentOrder) bool { return o.TargetUnit != nil && o.TargetUnit.InternalID == ids.enemy_unit }},
		{defs.OrderType_UnitAutoAttackBuilding, int64(ids.enemy_building), func(o *ai.CurrentOrder) bool {
			return o.TargetBuilding != nil && o.TargetBuilding.InternalID == ids.enemy_building
		}},
	} {
		order := decide(withUnitOrder(t, payload, c.order_type, c.target)).order
		if order == nil || !c.reached(order) {
			t.Errorf("order type %d with target %d produced %+v, want its target decoded", c.order_type, c.target, order)
		}
	}
}

type viewProbe struct {
	ai.Model
	Run func(view ai.View)
}

func (m *viewProbe) DecideOrders(view ai.View) ai.Decision {
	m.Run(view)
	return ai.Decision{}
}

func currentOrderAfter(t *testing.T, busy bool, issue func(view ai.View, u ai.UnitView)) *ai.CurrentOrder {
	t.Helper()
	g, player := bridgeGame(t, ownScout+`,{"x":3,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":11,"player":1,"count":1}]}]}`)
	var current *ai.CurrentOrder
	probe := &viewProbe{Model: ai.NoopModel{}, Run: func(view ai.View) {
		issue(view, view.Units()[0])
		current = view.Units()[0].CurrentOrder
	}}
	w := runWorker(t, player, probe, make(chan game.PlayerAction, 4))
	payload := gin.H(g.Risq.ToFrontend(uint64(g.Human(0)+1), false))
	if busy {
		payload = withUnitOrder(t, payload, defs.OrderType_UnitMoveSpace, int64(util.Pair(3, 0)))
	}
	player.AiUpdates <- &game.UpdateMessage{Kind: "start-turn", Content: gin.H{"game": payload}}
	receive(t, w.Actions)
	return current
}

var (
	spaceA = ai.Coordinate{X: 1, Y: 0}
	spaceB = ai.Coordinate{X: 2, Y: 0}
	busyAt = ai.Coordinate{X: 3, Y: 0}
)

var currentOrderCases = []struct {
	name  string
	busy  bool
	issue func(view ai.View, u ai.UnitView)
	want  *ai.Coordinate
}{
	{"replace with two verbs reports the first", false, func(v ai.View, u ai.UnitView) { v.MoveSpaceOrder(u, spaceA, true); v.MoveSpaceOrder(u, spaceB, false) }, &spaceA},
	{"append to an idle unit reports the first", false, func(v ai.View, u ai.UnitView) { v.MoveSpaceOrder(u, spaceA, false); v.MoveSpaceOrder(u, spaceB, false) }, &spaceA},
	{"append to a busy unit keeps its task", true, func(v ai.View, u ai.UnitView) { v.MoveSpaceOrder(u, spaceA, false) }, &busyAt},
	{"prepend reports the inserted task", true, func(v ai.View, u ai.UnitView) { v.MoveSpaceOrder(u, spaceA, true); v.MoveSpaceOrder(u, busyAt, false) }, &spaceA},
	{"append after a replace keeps the replacement", true, func(v ai.View, u ai.UnitView) { v.MoveSpaceOrder(u, spaceA, true); v.MoveSpaceOrder(u, spaceB, false) }, &spaceA},
	{"cancelling the old task keeps the replacement", true, func(v ai.View, u ai.UnitView) { v.MoveSpaceOrder(u, spaceA, true); v.CancelOrder(9999) }, &spaceA},
}

func TestCurrentOrderTracksTheQueueAcrossModes(t *testing.T) {
	for _, c := range currentOrderCases {
		t.Run(c.name, func(t *testing.T) {
			current := currentOrderAfter(t, c.busy, c.issue)
			if current == nil || current.TargetSpace == nil || *current.TargetSpace != *c.want {
				t.Errorf("current order = %+v, want a move to %+v", current, *c.want)
			}
		})
	}
}

func TestQueueKeepsInvisibleTargetsAndCancelsSharedOrders(t *testing.T) {
	g, player := bridgeGame(t, `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":21,"player":0,"count":2}]}]},{"x":3,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":11,"player":1,"count":1}]}]}`)
	payload := withUnitOrder(t, gin.H(g.Risq.ToFrontend(uint64(g.Human(0)+1), false)), defs.OrderType_UnitAttackUnit, 987654)
	probe := &viewProbe{Model: ai.NoopModel{}, Run: func(v ai.View) {
		entries := v.OrderQueue()
		if len(entries) != 2 {
			t.Fatalf("entries = %+v", entries)
		}
		for _, e := range entries {
			if e.TargetUID != 987654 || e.HasTargetLocation {
				t.Errorf("entry = %+v", e)
			}
		}
		if got := v.AssignedTo(false, 987654, nil); got != 2 {
			t.Errorf("assigned = %d", got)
		}
		v.CancelOrder(9999)
		if got := v.OrderQueue(); len(got) != 0 {
			t.Errorf("cancel left %+v", got)
		}
	}}
	w := runWorker(t, player, probe, make(chan game.PlayerAction, 4))
	player.AiUpdates <- &game.UpdateMessage{Kind: "start-turn", Content: gin.H{"game": payload}}
	receive(t, w.Actions)
}

type queueProbe struct {
	ai.Model
	Seen chan []ai.OrderEntry
}

func (m *queueProbe) DecideOrders(v ai.View) ai.Decision {
	decision := m.Model.DecideOrders(v)
	m.Seen <- v.OrderQueue()
	return decision
}

func queuePayload(g *harness.Game) gin.H {
	return gin.H{"game": g.Risq.ToFrontend(uint64(g.Human(0)+1), false)}
}

func TestQueueMirrorMatchesEngineReceipt(t *testing.T) {
	for _, mode := range []string{"replace", "append", "prepend", "cancel", "rejected"} {
		t.Run(mode, func(t *testing.T) {
			spaces := `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":21,"player":0,"count":2}]}]},{"x":1,"y":0,"terrain":1},{"x":2,"y":0,"terrain":1},{"x":3,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":11,"player":1,"count":1}]}]}`
			g, player := bridgeGame(t, spaces, func() {
				c := defs.UnitConfigs[21]
				c.Turn_stamina = 0
				c.Attack_type = defs.AttackType_NONE
				defs.UnitConfigs[21] = c
			})
			ids := []uint64{}
			for _, u := range g.Self(g.Human(0)).Units {
				ids = append(ids, u.InternalID)
			}
			g.Submit(g.Human(0), harness.OrderMove(ids, 2, 0))
			g.EndTurn()
			actions := `{"action":"select","mode":"` + mode + `","actors":{"kind":"units"},"pick":{"kind":"spaces","where":{"in":{"x":[1]}}},"issue":[{"move":{"to":"space"}}]}`
			if mode == "cancel" {
				actions = fmt.Sprintf(`{"action":"select","actors":{"kind":"units","where":{"in":{"uid":[%d]}}},"issue":[{"stop":{}}]}`, ids[0])
			}
			if mode == "rejected" {
				actions = `{"action":"select","actors":{"kind":"units"},"issue":[{"ungarrison":{}}]}`
			}
			probe := &queueProbe{Model: parsedModel(t, actions), Seen: make(chan []ai.OrderEntry, 1)}
			w := runWorker(t, player, probe, make(chan game.PlayerAction, 4))
			player.AiUpdates <- &game.UpdateMessage{Kind: "start-turn", Content: queuePayload(g)}
			predicted := receive(t, probe.Seen)
			orders := receive(t, w.Actions).Action["orders"].([]defs.OrderFromFrontend)
			g.Submit(g.Human(0), orders...)
			g.EndTurn()
			probe.Model = ai.NoopModel{}
			player.AiUpdates <- &game.UpdateMessage{Kind: "start-turn", Content: queuePayload(g)}
			actual := receive(t, probe.Seen)
			receive(t, w.Actions)
			if mode == "rejected" {
				if reflect.DeepEqual(predicted, actual) || len(g.Self(g.Human(0)).Refusals()) == 0 {
					t.Fatal("expected receipt rejection to differ from optimistic mirror")
				}
				return
			}
			if !reflect.DeepEqual(predicted, actual) {
				t.Fatalf("predicted %+v, engine %+v", predicted, actual)
			}
		})
	}
}
