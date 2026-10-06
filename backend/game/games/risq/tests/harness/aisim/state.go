package aisim

import (
	"encoding/json"
	"testing"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

func Decode(t *testing.T, payload gin.H) harness.State {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	var state harness.State
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatal(err)
	}
	return state
}

func (g *Game) State(t *testing.T) harness.State {
	return Decode(t, g.Risq.ToFrontend(0, true))
}

func Payload(t *testing.T, player *game.Player) gin.H {
	t.Helper()
	var payload gin.H
	player.ReplayUpdates(0, func(update *game.UpdateMessage) {
		if value, ok := update.Content["game"].(gin.H); ok {
			payload = value
		}
	})
	if payload == nil {
		t.Fatal("no retained game payload")
	}
	return payload
}

func (g *Game) Player(t *testing.T, slot int) *game.Player {
	t.Helper()
	for _, player := range g.Base.AiPlayers {
		if player.Player_id == slot {
			return player
		}
	}
	t.Fatalf("missing AI slot %d", slot)
	return nil
}

func (g *Game) Own(t *testing.T, slot int) harness.PlayerState {
	state := Decode(t, Payload(t, g.Player(t, slot)))
	return *state.Player(slot)
}
