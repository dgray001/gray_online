package orders

import (
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBuildingBehaviorUpdatesOnlyRequestedFields(t *testing.T) {
	g := orderGame(t, richBank, "")
	p := g.Human(0)
	id := buildingID(g, p, 23)
	g.Action(p, "set-building-behavior", gin.H{"internal_ids": []uint64{id}, "auto_attack": false, "interrupt_current": true, "target_priority": []uint8{0, 3, 255, 2}})
	b := entity(g, p, "buildings", id)
	if b["auto_attack"] != false || b["interrupt_current"] != true || !reflect.DeepEqual(b["target_priority"], []int{3, 2}) {
		t.Fatalf("behavior not applied: %v", b)
	}
	g.Action(p, "set-building-behavior", gin.H{"internal_ids": []uint64{id}, "auto_attack": true})
	b = entity(g, p, "buildings", id)
	if b["auto_attack"] != true || b["interrupt_current"] != true || !reflect.DeepEqual(b["target_priority"], []int{3, 2}) {
		t.Fatal("partial update reset omitted fields")
	}
	g.Action(p, "set-building-behavior", gin.H{"internal_ids": []uint64{id}, "target_priority": []uint8{}})
	if got := entity(g, p, "buildings", id)["target_priority"]; !reflect.DeepEqual(got, []int{}) {
		t.Fatalf("priority not cleared: %v", got)
	}
	if g.Self(p).OrdersSubmitted || len(g.Self(p).ActiveOrders) != 0 {
		t.Fatal("behavior action entered order pipeline")
	}
}
