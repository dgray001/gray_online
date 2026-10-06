package sims

import (
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/aisim"
	"reflect"
	"testing"
)

func assertBehaviorSequence(t *testing.T, g *aisim.Game, slot int) {
	t.Helper()
	var kinds []string
	for _, action := range g.Trace {
		if action.Ai_id == int(g.Player(t, slot).GetAiId()) {
			kinds = append(kinds, action.Kind)
		}
	}
	want := []string{"set-unit-behavior", "set-building-behavior", "submit-orders"}
	if !reflect.DeepEqual(kinds, want) {
		t.Fatalf("sequence=%v, want %v", kinds, want)
	}
	payload := aisim.Payload(t, g.Player(t, slot))
	assertBehaviorPayload(t, payload, slot)
}
