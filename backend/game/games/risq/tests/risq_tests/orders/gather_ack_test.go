package orders

import (
	"reflect"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

func TestGatherPointAcknowledgementsAreOwnerOnly(t *testing.T) {
	g := orderGame(t, richBank, "")
	p, other := g.Human(0), g.Human(1)
	id := buildingID(g, p, 1)
	g.Action(p, "set-gather-point", gin.H{"building_id": id, "clear": true})
	rawAction(g, p, "set-gather-point", gin.H{"building_id": id, "location_kind": 1, "location_id": harness.SpaceKey(3, 0)})
	update := nextUpdate(t, g.Base.Players[uint64(p+1)].Updates)
	if update.Kind != "gather-point-set" || update.Content["building_id"] != id || !reflect.DeepEqual(update.Content["gather_point"], entity(g, p, "buildings", id)["gather_point"]) {
		t.Fatalf("gather point acknowledgement differs from state: %+v", update)
	}
	rawAction(g, p, "set-gather-point", gin.H{"building_id": id, "clear": true})
	update = nextUpdate(t, g.Base.Players[uint64(p+1)].Updates)
	if update.Kind != "gather-point-set" || update.Content["building_id"] != id || update.Content["gather_point"] != nil {
		t.Fatalf("clear acknowledgement retained gather point: %+v", update)
	}
	if len(g.Base.Players[uint64(other+1)].Updates) != 0 || len(g.Base.ViewerUpdates) != 0 {
		t.Fatal("gather point notification reached another recipient")
	}
}
