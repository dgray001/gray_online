package orders

import (
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBehaviorIgnoresIneligibleForeignAndUnknownSubjects(t *testing.T) {
	g := orderGame(t, richBank, "")
	p, other := g.Human(0), g.Human(1)
	worker, foreign := unitIDs(g, p, 1)[0], unitIDs(g, other, 1)[0]
	beforeWorker, beforeForeign := entity(g, p, "units", worker), entity(g, other, "units", foreign)
	g.Action(p, "set-unit-behavior", gin.H{"internal_ids": []uint64{worker, foreign, 999999}, "stance": 2, "interrupt_current": true, "attack_back": false})
	if !reflect.DeepEqual(beforeWorker, entity(g, p, "units", worker)) || !reflect.DeepEqual(beforeForeign, entity(g, other, "units", foreign)) {
		t.Fatal("behavior changed ineligible or foreign unit")
	}
	unarmed, foreignBuilding := buildingID(g, p, 11), buildingID(g, other, 1)
	beforeBuilding, beforeOther := entity(g, p, "buildings", unarmed), entity(g, other, "buildings", foreignBuilding)
	g.Action(p, "set-building-behavior", gin.H{"internal_ids": []uint64{unarmed, foreignBuilding, 999999}, "auto_attack": false, "interrupt_current": true})
	if !reflect.DeepEqual(beforeBuilding, entity(g, p, "buildings", unarmed)) || !reflect.DeepEqual(beforeOther, entity(g, other, "buildings", foreignBuilding)) {
		t.Fatal("behavior changed unarmed or foreign building")
	}
}
