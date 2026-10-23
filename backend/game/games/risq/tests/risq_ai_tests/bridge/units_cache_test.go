package bridge

import (
	"encoding/json"
	"testing"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
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
