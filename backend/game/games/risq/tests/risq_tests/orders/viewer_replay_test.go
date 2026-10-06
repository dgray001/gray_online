package orders

import (
	"reflect"
	"testing"

	"github.com/dgray001/gray_online/game"
)

func TestViewerReplayReturnsLatestTurnSnapshot(t *testing.T) {
	g := orderGame(t, richBank, "")
	p := g.Human(0)
	viewer := game.CreateViewer(90, "viewer")
	g.Base.Viewers[90] = viewer
	latestUpdate(t, g.Base.ViewerUpdates)
	rawAction(g, p, "submit-orders", nil)
	first := nextUpdate(t, g.Base.ViewerUpdates)
	rawAction(g, p, "unsubmit-orders", nil)
	last := nextUpdate(t, g.Base.ViewerUpdates)
	if first.Id != 1 || last.Id != 1 || first.Kind != "submitted-orders" || last.Kind != "unsubmitted-orders" {
		t.Fatal("incorrect viewer updates")
	}
	for _, requested := range []int{-9, 1, 99} {
		g.Base.ResendViewerUpdate(90, requested)
		if got := nextUpdate(t, viewer.Updates); !reflect.DeepEqual(got, last) {
			t.Fatal("viewer replay did not return latest snapshot")
		}
	}
	g.Base.ResendLastUpdate(90)
	if got := nextUpdate(t, viewer.Updates); !reflect.DeepEqual(got, last) {
		t.Fatal("last viewer replay differs")
	}
	g.EndTurn()
	g.Base.ResendViewerUpdate(90, 1)
	if got := nextUpdate(t, viewer.Updates); got.Id != 1 || got.Kind != "start-turn" {
		t.Fatal("new turn did not replace viewer history")
	}
	g.Base.ResendPlayerUpdate(999999, 1)
	g.Base.ResendViewerUpdate(999999, 1)
	g.Base.ResendLastUpdate(999999)
	if len(viewer.Updates) != 0 {
		t.Fatal("unknown recipient received replay")
	}
}
