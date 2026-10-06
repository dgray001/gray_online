package orders

import (
	"reflect"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/gin-gonic/gin"
)

func TestUnitBehaviorUpdatesOnlyRequestedFields(t *testing.T) {
	g := orderGame(t, richBank, "")
	p := g.Human(0)
	id := unitIDs(g, p, 11)[0]
	g.Action(p, "set-unit-behavior", gin.H{"internal_ids": []uint64{id}, "stance": uint8(defs.UnitStance_AGGRESSIVE), "interrupt_current": true, "attack_back": false, "target_priority": []uint8{0, 3, 255, 2}})
	u := entity(g, p, "units", id)
	if u["stance"] != defs.UnitStance_AGGRESSIVE || u["interrupt_current"] != true || u["attack_back"] != false || !reflect.DeepEqual(u["target_priority"], []int{3, 2}) {
		t.Fatalf("behavior not applied: %v", u)
	}
	g.Action(p, "set-unit-behavior", gin.H{"internal_ids": []uint64{id}, "attack_back": true})
	u = entity(g, p, "units", id)
	if u["stance"] != defs.UnitStance_AGGRESSIVE || u["interrupt_current"] != true || u["attack_back"] != true || !reflect.DeepEqual(u["target_priority"], []int{3, 2}) {
		t.Fatal("partial behavior update reset omitted fields")
	}
	for _, stance := range []uint8{0, uint8(defs.UnitStance_END), 255} {
		g.Action(p, "set-unit-behavior", gin.H{"internal_ids": []uint64{id}, "stance": stance})
		if entity(g, p, "units", id)["stance"] != defs.UnitStance_AGGRESSIVE {
			t.Fatal("invalid stance changed unit")
		}
	}
	g.Action(p, "set-unit-behavior", gin.H{"internal_ids": []uint64{id}, "target_priority": []uint8{}})
	if got := entity(g, p, "units", id)["target_priority"]; !reflect.DeepEqual(got, []int{}) {
		t.Fatalf("priority not cleared: %v", got)
	}
	if g.Self(p).OrdersSubmitted || len(g.Self(p).ActiveOrders) != 0 {
		t.Fatal("behavior action entered order pipeline")
	}
}
