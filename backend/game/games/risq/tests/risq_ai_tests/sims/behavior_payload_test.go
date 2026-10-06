package sims

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"testing"
)

func assertBehaviorPayload(t *testing.T, payload gin.H, slot int) {
	t.Helper()
	var state struct {
		Players []struct {
			Player struct {
				ID int `json:"player_id"`
			} `json:"player"`
			Units []struct {
				Stance     uint8 `json:"stance"`
				AttackBack bool  `json:"attack_back"`
			} `json:"units"`
			Buildings []struct {
				AutoAttack bool `json:"auto_attack"`
			} `json:"buildings"`
		} `json:"players"`
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatal(err)
	}
	for _, player := range state.Players {
		if player.Player.ID != slot {
			continue
		}
		if len(player.Units) != 1 || player.Units[0].Stance != 1 || player.Units[0].AttackBack {
			t.Fatalf("unit behavior=%+v", player.Units)
		}
		if len(player.Buildings) != 1 || player.Buildings[0].AutoAttack {
			t.Fatalf("building behavior=%+v", player.Buildings)
		}
		return
	}
	t.Fatalf("missing player %d", slot)
}
