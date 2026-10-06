package orders

import (
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestPlayerReplayReturnsLatestSnapshot(t *testing.T) {
	g := orderGame(t, richBank, "")
	p, other := g.Human(0), g.Human(1)
	player := g.Base.Players[uint64(p+1)]
	latestUpdate(t, player.Updates)
	g.Submit(p)
	rawAction(g, p, "set-unit-behavior", gin.H{})
	if failures := g.Failed(p); len(failures) != 1 {
		t.Fatal("setup did not reject submitted action")
	}
	g.Base.ResendLastUpdate(uint64(p + 1))
	before := nextUpdate(t, player.Updates)
	if before.Id != 1 || before.Kind != "submitted-orders" {
		t.Fatalf("failed action replaced replay: %+v", before)
	}
	g.Action(p, "unsubmit-orders", nil)
	rawAction(g, p, "set-unit-behavior", gin.H{"internal_ids": unitIDs(g, p, 11), "stance": 1})
	first := nextUpdate(t, player.Updates)
	rawAction(g, p, "set-unit-behavior", gin.H{"internal_ids": unitIDs(g, p, 11), "stance": 2})
	last := nextUpdate(t, player.Updates)
	if first.Id != 1 || last.Id != 1 || last.Kind != "unit-behavior-set" || reflect.DeepEqual(first.Content, last.Content) {
		t.Fatal("snapshot updates have wrong IDs or content")
	}
	for _, requested := range []int{-9, 1, 99} {
		g.Base.ResendPlayerUpdate(uint64(p+1), requested)
		if got := nextUpdate(t, player.Updates); !reflect.DeepEqual(got, last) {
			t.Fatalf("replay %d returned %+v, want %+v", requested, got, last)
		}
	}
	g.Base.ResendLastUpdate(uint64(p + 1))
	if got := nextUpdate(t, player.Updates); !reflect.DeepEqual(got, last) {
		t.Fatal("last replay differs")
	}
	g.Base.ResendLastUpdate(uint64(other + 1))
	if got := nextUpdate(t, g.Base.Players[uint64(other+1)].Updates); got.Kind != "unsubmitted-orders" {
		t.Fatal("recipient histories were mixed")
	}
}
