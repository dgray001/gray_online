package orders

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

func TestGatherPointFoundationTargets(t *testing.T) {
	for _, planned := range []bool{true, false} {
		for _, target := range []string{"site-point", "object-point", "zone-point"} {
			explicit := target != "zone-point"
			name := map[bool]string{true: "planned", false: "materialized"}[planned] + "/" + target
			t.Run(name, func(t *testing.T) {
				g := orderGame(t, richBank, "")
				p, x := g.Human(0), 0
				if planned {
					x = 3
				}
				config := defs.BuildingConfigs[2]
				config.Build_stamina = 10000
				defs.BuildingConfigs[2] = config
				g.Submit(p, harness.OrderBuild(unitIDs(g, p, 1)[:1], 2, x, 0, -1, 1))
				g.EndTurn()
				order := g.Self(p).ActiveOrders[0].InternalID
				g.Submit(p, harness.Order(defs.OrderType_CancelOrder, nil, int64(order), false))
				g.EndTurn()
				point := gin.H{"location_kind": 2, "location_id": harness.ZoneKey(x, 0, -1, 1)}
				if explicit {
					point["object_type"], point["object_id"] = 2, 0
				}
				if target == "object-point" && !planned {
					point["object_type"], point["object_id"] = 2, buildingID(g, p, 2)
				}
				u := produceAtPoint(g, p, 1, 1, point)
				building := len(u.ActiveOrders) == 1 && u.ActiveOrders[0].OrderType == uint8(defs.OrderType_UnitBuild)
				if building != explicit {
					t.Fatalf("build order=%t, want %t: %+v", building, explicit, u.ActiveOrders)
				}
			})
		}
	}
}

func TestGatherPointFollowsSameTurnConstruction(t *testing.T) {
	g := orderGame(t, richBank, "")
	p := g.Human(0)
	config := defs.BuildingConfigs[2]
	config.Build_stamina = 10000
	defs.BuildingConfigs[2] = config
	producer := buildingID(g, p, 1)
	workers := unitIDs(g, p, 1)
	g.Action(p, "set-gather-point", gin.H{"building_id": producer, "location_kind": 2, "location_id": harness.ZoneKey(0, 0, -1, 1), "object_type": 2, "object_id": 0})
	g.Submit(p, harness.OrderBuild(workers[:1], 2, 0, 0, -1, 1), harness.OrderProduce([]uint64{producer}, 1))
	g.EndTurn()
	units := unitIDs(g, p, 1)
	if len(units) != len(workers)+1 {
		t.Fatal("villager was not produced")
	}
	u := g.State(p).Unit(units[len(units)-1])
	if len(u.ActiveOrders) != 1 || u.ActiveOrders[0].OrderType != uint8(defs.OrderType_UnitBuild) {
		t.Fatalf("site point lost materialized foundation: %+v", u.ActiveOrders)
	}
}

func TestMissingPlannedFoundationGatherPointMoves(t *testing.T) {
	for _, kind := range []uint32{1, 11} {
		g := orderGame(t, richBank, "")
		p, producer := g.Human(0), uint32(1)
		if kind == 11 {
			producer = 22
		}
		u := produceAtPoint(g, p, producer, kind, gin.H{"location_kind": 2, "location_id": harness.ZoneKey(0, 0, -1, 1), "object_type": 2, "object_id": 0})
		g.EndTurn()
		u = *g.State(p).Unit(u.InternalID)
		if u.Zone != (harness.Coord{X: -1, Y: 1}) || len(u.ActiveOrders) != 0 || len(g.Self(p).PlannedFoundations) != 0 {
			t.Fatalf("missing foundation should mean movement: %+v", u)
		}
	}
}
