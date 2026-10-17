package bridge

import (
	"encoding/json"
	"testing"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/risq/ai"
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

func TestUnitsCacheSurvivesDecisionAndInvalidatesOnAssign(t *testing.T) {
	spaces := ownScout + `,{"x":1,"y":0,"terrain":1,"zones":[{"x":0,"y":0}]},{"x":3,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":1,"player":1,"count":1}]}]}`
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
