package orders

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBehaviorNotificationsRespectSpyVisibility(t *testing.T) {
	for _, spy := range []bool{false, true} {
		t.Run(fmt.Sprint(spy), func(t *testing.T) {
			g := notificationGame(t, spy)
			p, other := g.Human(0), g.Human(1)
			g.Submit(p)
			g.Action(p, "unsubmit-orders", nil)
			var visible, hidden uint64
			for _, u := range g.Self(p).Units {
				if u.Space.X == 0 {
					visible = u.InternalID
				} else {
					hidden = u.InternalID
				}
			}
			for _, kind := range []string{"unit", "building"} {
				if kind == "building" {
					visible, hidden = buildingID(g, p, 23), buildingID(g, p, 21)
				}
				ids := []uint64{visible, hidden}
				rawAction(g, p, "set-"+kind+"-behavior", gin.H{"internal_ids": append(append([]uint64(nil), ids...), 999999), "stance": 2, "auto_attack": false, "interrupt_current": true, "target_priority": []uint8{0, 3, 255, 2}})
				owner := nextUpdate(t, g.Base.Players[uint64(p+1)].Updates)
				if owner.Kind != kind+"-behavior-set" || !reflect.DeepEqual(owner.Content["internal_ids"], ids) || !reflect.DeepEqual(owner.Content["target_priority"], []int{3, 2}) {
					t.Fatalf("owner notification: %+v", owner)
				}
				updates := g.Base.Players[uint64(other+1)].Updates
				if spy {
					observer := nextUpdate(t, updates)
					if observer.Kind != owner.Kind || !reflect.DeepEqual(observer.Content["internal_ids"], []uint64{visible}) || !reflect.DeepEqual(observer.Content["target_priority"], []int{3, 2}) {
						t.Fatalf("observer notification leaked subjects: %+v", observer)
					}
				} else if len(updates) != 0 {
					t.Fatal("ordinary vision received private behavior notification")
				}
			}
		})
	}
}
