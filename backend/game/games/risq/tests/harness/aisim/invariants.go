package aisim

import (
	"math"
	"testing"
)

func (g *Game) Check(t *testing.T) {
	t.Helper()
	locations := make(map[uint64]int)
	for slot := range len(g.Base.AiPlayers) {
		for _, row := range Decode(t, Payload(t, g.Player(t, slot))).Spaces {
			for _, space := range row {
				if space == nil {
					continue
				}
				for _, zones := range space.Zones {
					for _, zone := range zones {
						for _, unit := range zone.Units {
							if unit.PlayerID == slot {
								locations[unit.InternalID]++
							}
						}
					}
				}
			}
		}
	}
	for slot := range len(g.Base.AiPlayers) {
		own := g.Own(t, slot)
		for _, value := range []float64{own.Resources.Food, own.Resources.Wood, own.Resources.Stone, own.Resources.Gold} {
			if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
				t.Fatalf("slot %d balance=%+v", slot, own.Resources)
			}
		}
		if len(own.Units) > own.PopulationLimit {
			t.Fatalf("slot %d population %d exceeds %d", slot, len(own.Units), own.PopulationLimit)
		}
		for _, unit := range own.Units {
			if unit.GarrisonedIn != nil {
				locations[unit.InternalID]++
			}
			if locations[unit.InternalID] != 1 || unit.PlayerID != slot {
				t.Fatalf("unit location/owner=%+v count=%d", unit, locations[unit.InternalID])
			}
			if unit.CombatStats.Health <= 0 || unit.CombatStats.Health > float64(unit.CombatStats.MaxHealth) || math.IsNaN(unit.CombatStats.Health) {
				t.Fatalf("health=%+v", unit)
			}
			if unit.CurrentStamina < 0 || unit.CurrentStamina > 2*unit.TurnStamina {
				t.Fatalf("stamina=%+v", unit)
			}
		}
		for _, building := range own.Buildings {
			if building.PlayerID != slot || building.CombatStats.Health <= 0 || building.CombatStats.Health > float64(building.CombatStats.MaxHealth) || math.IsNaN(building.CombatStats.Health) {
				t.Fatalf("building=%+v", building)
			}
		}
	}
}
