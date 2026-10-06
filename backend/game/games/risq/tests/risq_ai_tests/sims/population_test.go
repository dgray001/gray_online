package sims

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"testing"
)

func TestHireAndProductionSharePopulationReservation(t *testing.T) {
	for _, hireFirst := range []bool{true, false} {
		t.Run(map[bool]string{true: "hire-first", false: "create-first"}[hireFirst], func(t *testing.T) {
			hire, create := `{"action":"hire","unit_id":11,"max":2}`, `{"action":"create","unit_id":1,"queue":2}`
			actions := create + "," + hire
			if hireFirst {
				actions = hire + "," + create
			}
			spaces := `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":1,"player":0},"units":[{"id":1,"player":0,"count":4}]}]},` + remote
			g := fixture(t, `{"food":100,"gold":500}`, spaces, `,"starting_techs":[4]`, once(actions), emptyModel)
			g.Run(t, 2)
			p := g.Own(t, 0)
			if len(p.Units) != 5 {
				t.Fatalf("population=%+v", p.Units)
			}
			creates, hires, food, gold := 1, 0, 50.0, 500.0
			if hireFirst {
				creates, hires, food, gold = 0, 1, 100, 409
			}
			assertOrderCount(t, g, 0, defs.OrderType_BuildingCreate, creates)
			assertOrderCount(t, g, 0, defs.OrderType_BuyMercenary, hires)
			if p.Resources.Food != food || p.Resources.Gold != gold {
				t.Fatalf("balance=%+v", p.Resources)
			}
		})
	}
}
