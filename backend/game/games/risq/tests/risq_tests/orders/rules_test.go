package orders

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/risq"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

type validator func(typ defs.OrderType) defs.OrderFromFrontend

// A well-formed order of every type, built from ids resolved once
func validOrders(mine, foe roster, produceOrder uint64) validator {
	return func(typ defs.OrderType) defs.OrderFromFrontend {
		u, center, target := mine.workers[0], mine.buildings[201], int64(0)
		subjects := []uint64{u}
		if typ.IsBuildingOrder() {
			subjects = []uint64{center}
		} else if typ.IsPlayerOrder() {
			subjects = nil
		}
		if typ.IsAttackOrder() && typ.IsBuildingOrder() {
			subjects = []uint64{mine.buildings[208]}
		} else if typ.IsAttackOrder() {
			subjects = mine.grunt[:1]
		}
		switch typ {
		case defs.OrderType_UnitMoveSpace, defs.OrderType_UnitAttackSpace:
			target = harness.SpaceKey(1, 0)
		case defs.OrderType_UnitMoveZone, defs.OrderType_UnitAttackZone:
			target = harness.ZoneKey(1, 0, 0, 0)
		case defs.OrderType_UnitGather:
			target = harness.ZoneKey(0, 0, 1, -1)
		case defs.OrderType_UnitBuild:
			target = harness.BuildKey(202, 1, 0, 0, 0)
		case defs.OrderType_UnitRepair, defs.OrderType_UnitGarrison:
			target = int64(center)
		case defs.OrderType_UnitRenew:
			target = int64(mine.buildings[203])
		case defs.OrderType_UnitAttackUnit, defs.OrderType_BuildingAttackUnit:
			target = int64(foe.workers[0])
		case defs.OrderType_UnitAttackBuilding, defs.OrderType_BuildingAttackBuilding:
			target = int64(foe.buildings[201])
		case defs.OrderType_BuildingCreate:
			target = 11
		case defs.OrderType_BuildingResearch:
			target = 301
		case defs.OrderType_BuyMercenary:
			target = harness.BuildKey(21, 0, 0, 0, 0)
		case defs.OrderType_CancelOrder:
			target = int64(produceOrder)
		case defs.OrderType_CancelFoundation:
			target = harness.ZoneKey(3, 0, 0, 0)
		}
		return harness.Order(typ, subjects, target, false)
	}
}

func blockedActions(g *harness.Game, p int, mine roster) map[string]gin.H {
	order := harness.OrderMove(mine.workers[:1], 2, 0)
	order.Player_id = g.PlayerID(p)
	return map[string]gin.H{
		"submit-orders":         {"orders": []defs.OrderFromFrontend{order}},
		"set-unit-behavior":     {"internal_ids": mine.grunt, "stance": 2, "attack_back": false, "interrupt_current": true},
		"set-building-behavior": {"internal_ids": []uint64{mine.buildings[208]}, "auto_attack": false, "interrupt_current": true},
		"set-gather-point":      {"building_id": mine.buildings[201], "location_kind": 1, "location_id": harness.SpaceKey(3, 0)},
	}
}

// Validation, gating, behavior and gather point bookkeeping that never need more than a few turns
func TestOrderRules(t *testing.T) {
	g := newGame(t, layout{workers: 4, grunt: 2, extra: `,"starting_techs":[302]`})
	p, e := g.Human(0), g.Human(1)
	mine, foe := scan(g, p), scan(g, e)
	v, inf, b := mine.workers, mine.grunt, mine.buildings
	research := harness.Order(defs.OrderType_BuildingResearch, []uint64{b[209]}, 303, false)

	g.Submit(p, harness.OrderProduce([]uint64{b[205]}, 31), harness.OrderBuild(v[:1], 202, 3, 0, 0, 0), harness.OrderBuild(v[1:2], 202, 0, 0, -1, 1), research)
	g.EndTurn()
	var unfinished uint64
	for _, building := range g.Self(p).Buildings {
		if building.BuildingID == 202 && building.UnderConstruction {
			unfinished = building.InternalID
		}
	}
	produced := activeOrderIDs(g, p, uint8(defs.OrderType_BuildingCreate))
	if unfinished == 0 || len(produced) != 1 || len(g.Self(p).PlannedFoundations) != 1 {
		t.Fatalf("setup left no unfinished building, production order and planned foundation: %+v", g.Self(p))
	}
	valid := validOrders(mine, foe, produced[0])

	// every order type is either accepted whole or rejected for a bad subject
	for typ := defs.OrderType_None; typ <= defs.OrderType_END; typ++ {
		order := valid(typ)
		if typ == defs.OrderType_None || typ == defs.OrderType_END || typ.IsAutoSynthesized() {
			g.SubmitRejected(p, order, fmt.Sprintf("Invalid order type: %d", typ))
			continue
		}
		before := g.Risq.StateHash()
		g.Submit(p, order)
		if g.Risq.StateHash() == before {
			t.Fatalf("valid order %d did not mark submitted", typ)
		}
		g.Action(p, "unsubmit-orders", nil)
		order.Subjects = []uint64{999999}
		message := "Invalid subjects in player order"
		if typ.IsUnitOrder() {
			message = "Invalid unit subject id"
		}
		if typ.IsBuildingOrder() {
			message = "Invalid building subject id"
		}
		if typ == defs.OrderType_CancelOrder {
			message = fmt.Sprintf("Subject 999999 is not on order %d", order.Target_id)
		}
		g.SubmitRejected(p, order, message)
	}

	unchanged(g, "subject restrictions", func() {
		for typ := defs.OrderType_UnitMoveSpace; typ < defs.OrderType_END; typ++ {
			if typ.IsAutoSynthesized() || (!typ.IsUnitOrder() && !typ.IsBuildingOrder()) {
				continue
			}
			order := valid(typ)
			order.Subjects = nil
			g.SubmitRejected(p, order, "Order must have at least one subject")
			foreign, owned, message := foe.workers[0], v[0], "Invalid unit subject id"
			if typ.IsBuildingOrder() {
				foreign, owned, message = foe.buildings[201], b[201], "Invalid building subject id"
			}
			if typ.IsAttackOrder() {
				owned = valid(typ).Subjects[0]
			}
			order.Subjects = []uint64{foreign}
			g.SubmitRejected(p, order, message)
			order.Player_id = g.PlayerID(p)
			for _, invalid := range []uint64{foreign, 999999} {
				for _, subjects := range [][]uint64{{owned, invalid}, {invalid, owned}} {
					order.Subjects = subjects
					rejectAction(g, p, "submit-orders", gin.H{"orders": []defs.OrderFromFrontend{order}}, message)
				}
			}
		}
	})

	unchanged(g, "invalid targets", func() {
		cases := []struct {
			typ     defs.OrderType
			target  int64
			message string
		}{
			{defs.OrderType_UnitMoveSpace, harness.SpaceKey(9, 0), "Invalid space target inverted from"},
			{defs.OrderType_UnitMoveZone, harness.ZoneKey(9, 0, 0, 0), "Invalid space target inverted from zone key"},
			{defs.OrderType_UnitMoveZone, harness.ZoneKey(0, 0, 9, 0), "Invalid zone target inverted from zone key"},
			{defs.OrderType_UnitGather, harness.ZoneKey(2, 0, 0, 0), "No resource in target zone"},
			{defs.OrderType_UnitGather, harness.ZoneKey(9, 0, 0, 0), "Invalid space target inverted from zone key"},
			{defs.OrderType_UnitGather, harness.ZoneKey(0, 0, 9, 0), "Invalid zone target inverted from zone key"},
			{defs.OrderType_UnitBuild, harness.BuildKey(999, 1, 0, 0, 0), "Invalid or unbuildable building id:"},
			{defs.OrderType_UnitBuild, harness.BuildKey(202, 9, 0, 0, 0), "Invalid space or zone target inverted from build key"},
			{defs.OrderType_UnitBuild, harness.BuildKey(202, 0, 0, 1, -1), "Target zone is already occupied"},
			{defs.OrderType_UnitBuild, harness.BuildKey(202, 0, 0, 0, 0), "Target zone is already occupied"},
			{defs.OrderType_UnitBuild, harness.BuildKey(203, 1, 0, 0, 0), "Required tech id 301 not researched"},
			{defs.OrderType_UnitRepair, 999999, "Invalid building target id"},
			{defs.OrderType_UnitRenew, 999999, "Invalid building target id"},
			{defs.OrderType_UnitAttackUnit, 999999, "Invalid unit target id"},
			{defs.OrderType_UnitAttackBuilding, 999999, "Invalid building target id"},
			{defs.OrderType_UnitAttackZone, harness.ZoneKey(9, 0, 0, 0), "Invalid space target inverted from zone key"},
			{defs.OrderType_UnitAttackZone, harness.ZoneKey(0, 0, 9, 0), "Invalid zone target inverted from zone key"},
			{defs.OrderType_UnitAttackSpace, harness.SpaceKey(9, 0), "Invalid space target inverted from"},
			{defs.OrderType_UnitGarrison, 999999, "Invalid building target id"},
			{defs.OrderType_BuildingCreate, 999999, "Invalid or unsupported unit id for production:"},
			{defs.OrderType_BuildingResearch, 999999, "Invalid or unsupported tech id:"},
			{defs.OrderType_BuildingAttackUnit, 999999, "Invalid unit target id"},
			{defs.OrderType_BuildingAttackBuilding, 999999, "Invalid building target id"},
			{defs.OrderType_CancelOrder, 999999, "No active order with id"},
			{defs.OrderType_CancelFoundation, harness.ZoneKey(9, 0, 0, 0), "Invalid zone target inverted from zone key"},
			{defs.OrderType_CancelFoundation, harness.ZoneKey(1, 0, 0, 0), "No planned foundation at this zone"},
			{defs.OrderType_BuyMercenary, harness.BuildKey(21, 9, 0, 0, 0), "Invalid space or zone target inverted from mercenary key"},
			{defs.OrderType_BuyMercenary, harness.BuildKey(999, 0, 0, 0, 0), "Invalid or unsupported unit id:"},
			{defs.OrderType_BuyMercenary, harness.BuildKey(22, 0, 0, 0, 0), "not an available mercenary"},
		}
		for _, c := range cases {
			order := valid(c.typ)
			order.Player_id, order.Target_id = g.PlayerID(p), c.target
			rejectAction(g, p, "submit-orders", gin.H{"orders": []defs.OrderFromFrontend{order}}, c.message)
		}
	})

	// a batch with any ineligible subject is rejected whole, whichever position it holds
	unchanged(g, "mixed eligibility", func() {
		for _, c := range []struct {
			typ     defs.OrderType
			message string
		}{
			{defs.OrderType_UnitGather, "Only economic units can gather"},
			{defs.OrderType_UnitBuild, "Only economic units can build"},
			{defs.OrderType_UnitRepair, "Only economic units can repair"},
			{defs.OrderType_UnitRenew, "Only economic units can renew"},
			{defs.OrderType_BuildingCreate, "Building id 205 cannot produce unit id 11"},
			{defs.OrderType_BuildingResearch, "Building id 205 cannot research tech id 301"},
			{defs.OrderType_UnitAttackSpace, "Unit id %d cannot attack"},
			{defs.OrderType_UnitAttackZone, "Unit id %d cannot attack"},
			{defs.OrderType_UnitAttackUnit, "Unit id %d cannot attack"},
			{defs.OrderType_UnitAttackBuilding, "Unit id %d cannot attack"},
			{defs.OrderType_BuildingAttackUnit, "Building id %d cannot attack"},
			{defs.OrderType_BuildingAttackBuilding, "Building id %d cannot attack"},
		} {
			order := valid(c.typ)
			eligible, ineligible := order.Subjects[0], inf[0]
			if c.typ.IsBuildingOrder() {
				ineligible = b[205]
			} else if c.typ.IsAttackOrder() {
				ineligible = v[0]
			}
			message := c.message
			if strings.Contains(message, "%d") {
				message = fmt.Sprintf(message, ineligible)
			}
			order.Player_id = g.PlayerID(p)
			for _, subjects := range [][]uint64{{eligible, ineligible}, {ineligible, eligible}} {
				order.Subjects = subjects
				rejectAction(g, p, "submit-orders", gin.H{"orders": []defs.OrderFromFrontend{order}}, message)
			}
		}
		for _, c := range []struct {
			typ     defs.OrderType
			message string
		}{
			{defs.OrderType_UnitGather, "Only economic units can gather"}, {defs.OrderType_UnitBuild, "Only economic units can build"},
			{defs.OrderType_UnitRepair, "Only economic units can repair"}, {defs.OrderType_UnitRenew, "Only economic units can renew"},
		} {
			order := valid(c.typ)
			order.Subjects = inf
			g.SubmitRejected(p, order, c.message)
		}
	})

	g.SubmitRejected(p, harness.OrderProduce([]uint64{b[201]}, 21), "Building id 201 cannot produce unit id 21")
	g.SubmitRejected(p, harness.Order(defs.OrderType_BuildingResearch, []uint64{b[201]}, 304, false), "Building id 201 cannot research tech id 304")
	g.SubmitRejected(p, harness.Order(defs.OrderType_BuildingAttackUnit, []uint64{b[209]}, int64(foe.workers[0]), false), fmt.Sprintf("Building id %d cannot attack", b[209]))

	unchanged(g, "malformed actions", func() {
		for kind, payload := range map[string]gin.H{
			"submit-orders":         {"orders": "invalid"},
			"set-unit-behavior":     {"internal_ids": "invalid"},
			"set-building-behavior": {"auto_attack": "invalid"},
			"set-gather-point":      {"building_id": "invalid"},
		} {
			rejectAction(g, p, kind, payload, "")
		}
	})

	for typ, target := range map[defs.OrderType]int64{defs.OrderType_BuildingCreate: 11, defs.OrderType_BuildingResearch: 301} {
		g.SubmitRejected(p, harness.Order(typ, []uint64{unfinished}, target, false), fmt.Sprintf("Building id %d is still under construction", unfinished))
	}
	g.SubmitRejected(p, research, "Tech id 303 is already being researched")

	// ownership of the submission and atomic validation
	move := harness.OrderMove(v[2:3], 1, 0)
	move.Player_id = g.PlayerID(e)
	unchanged(g, "identity and atomic validation", func() {
		rejectAction(g, p, "submit-orders", gin.H{"orders": []defs.OrderFromFrontend{move}}, fmt.Sprintf("Order player id %d does not match submitter %d", move.Player_id, g.PlayerID(p)))
		move.Player_id = g.PlayerID(p)
		bad := move
		bad.Subjects = []uint64{999999}
		rejectAction(g, p, "submit-orders", gin.H{"orders": []defs.OrderFromFrontend{move, bad}}, "Invalid unit subject id")
		rejectAction(g, p, "unsubmit-orders", nil, "Orders not submitted")
	})

	// a submitted player cannot change anything until they unsubmit
	g.Submit(p, move)
	unchanged(g, "submitted-player actions", func() {
		for kind, payload := range blockedActions(g, p, mine) {
			rejectAction(g, p, kind, payload, "Orders already submitted")
		}
	})
	g.EndTurn()
	if u := unitByID(g, p, v[2]); u.Space != (harness.Coord{X: 1}) || len(u.ActiveOrders) != 0 {
		t.Fatalf("valid resubmission did not finish: %+v", u)
	}
	g.Submit(p, harness.OrderMove(v[3:4], 1, 0))
	g.Action(p, "unsubmit-orders", nil)
	g.Action(p, "set-unit-behavior", gin.H{"internal_ids": inf, "stance": 1})
	if g.Self(p).OrdersSubmitted {
		t.Fatal("unsubmit did not reopen actions")
	}

	// behaviors apply only the fields given, and never touch foreign, unknown or ineligible subjects
	unit := inf[0]
	g.Action(p, "set-unit-behavior", gin.H{"internal_ids": []uint64{unit}, "stance": uint8(defs.UnitStance_AGGRESSIVE), "interrupt_current": true, "attack_back": false, "target_priority": []uint8{0, 3, 255, 2}})
	if u := entity(g, p, "units", unit); u["stance"] != defs.UnitStance_AGGRESSIVE || u["interrupt_current"] != true || u["attack_back"] != false || !reflect.DeepEqual(u["target_priority"], []int{3, 2}) {
		t.Fatalf("behavior not applied: %v", u)
	}
	g.Action(p, "set-unit-behavior", gin.H{"internal_ids": []uint64{unit}, "attack_back": true})
	if u := entity(g, p, "units", unit); u["stance"] != defs.UnitStance_AGGRESSIVE || u["interrupt_current"] != true || u["attack_back"] != true || !reflect.DeepEqual(u["target_priority"], []int{3, 2}) {
		t.Fatal("partial behavior update reset omitted fields")
	}
	unchanged(g, "invalid stances", func() {
		for _, stance := range []uint8{0, uint8(defs.UnitStance_END), 255} {
			g.Action(p, "set-unit-behavior", gin.H{"internal_ids": []uint64{unit}, "stance": stance})
		}
	})
	g.Action(p, "set-unit-behavior", gin.H{"internal_ids": []uint64{unit}, "target_priority": []uint8{}})
	if got := entity(g, p, "units", unit)["target_priority"]; !reflect.DeepEqual(got, []int{}) {
		t.Fatalf("priority not cleared: %v", got)
	}
	fortress := b[208]
	g.Action(p, "set-building-behavior", gin.H{"internal_ids": []uint64{fortress}, "auto_attack": false, "interrupt_current": true, "target_priority": []uint8{0, 3, 255, 2}})
	if got := entity(g, p, "buildings", fortress); got["auto_attack"] != false || got["interrupt_current"] != true || !reflect.DeepEqual(got["target_priority"], []int{3, 2}) {
		t.Fatalf("building behavior not applied: %v", got)
	}
	g.Action(p, "set-building-behavior", gin.H{"internal_ids": []uint64{fortress}, "auto_attack": true})
	if got := entity(g, p, "buildings", fortress); got["auto_attack"] != true || got["interrupt_current"] != true || !reflect.DeepEqual(got["target_priority"], []int{3, 2}) {
		t.Fatal("partial building update reset omitted fields")
	}
	g.Action(p, "set-building-behavior", gin.H{"internal_ids": []uint64{fortress}, "target_priority": []uint8{}})
	if got := entity(g, p, "buildings", fortress)["target_priority"]; !reflect.DeepEqual(got, []int{}) {
		t.Fatalf("building priority not cleared: %v", got)
	}
	unchanged(g, "ineligible behavior subjects", func() {
		g.Action(p, "set-unit-behavior", gin.H{"internal_ids": []uint64{v[0], foe.workers[0], 999999}, "stance": 2, "interrupt_current": true, "attack_back": false})
		g.Action(p, "set-building-behavior", gin.H{"internal_ids": []uint64{b[209], foe.buildings[201], 999999}, "auto_attack": false, "interrupt_current": true})
	})
	if g.Self(p).OrdersSubmitted {
		t.Fatal("behavior actions entered the order pipeline")
	}

	// gather points: set, replace, clear, and refuse invalid ones without changing the existing point
	center := b[201]
	g.Action(p, "set-gather-point", gin.H{"building_id": center, "location_kind": 1, "location_id": harness.SpaceKey(3, 0)})
	if gp := entity(g, p, "buildings", center)["gather_point"].(gin.H); gp["location_kind"] != risq.RisqGatherPointLocationKind_SPACE || gp["location_id"] != uint64(harness.SpaceKey(3, 0)) {
		t.Fatalf("space gather point: %v", gp)
	}
	unchanged(g, "invalid gather points", func() {
		for _, payload := range []gin.H{
			{"location_kind": 0}, {"location_kind": 3}, {"location_kind": 255},
			{"location_kind": 1, "location_id": harness.SpaceKey(9, 0)},
			{"location_kind": 2, "location_id": harness.ZoneKey(0, 0, 9, 0)},
			{"location_kind": 2, "location_id": harness.ZoneKey(9, 0, 0, 0)},
			{"location_kind": 1, "object_type": 4}, {"location_kind": 1, "object_type": 255},
		} {
			payload["building_id"] = center
			rejectAction(g, p, "set-gather-point", payload, "gather point")
		}
		for _, subject := range []uint64{999999, foe.buildings[201], b[209]} {
			rejectAction(g, p, "set-gather-point", gin.H{"building_id": subject, "location_kind": 1, "location_id": harness.SpaceKey(3, 0)}, "gather point")
		}
	})
	zone := harness.ZoneKey(0, 0, 1, -1)
	g.Action(p, "set-gather-point", gin.H{"building_id": center, "location_kind": 2, "location_id": zone, "object_type": 3, "object_id": 11})
	if gp := entity(g, p, "buildings", center)["gather_point"].(gin.H); gp["location_kind"] != risq.RisqGatherPointLocationKind_ZONE || gp["location_id"] != uint64(zone) || gp["object_type"] != risq.RisqGatherObjectType_RESOURCE || gp["object_id"] != uint64(11) {
		t.Fatalf("zone gather point: %v", gp)
	}
	g.Action(p, "set-gather-point", gin.H{"building_id": center, "clear": true})
	if _, present := entity(g, p, "buildings", center)["gather_point"]; present {
		t.Fatal("gather point survived clear")
	}

	unchanged(g, "auto-renew failures", func() {
		for _, action := range []gin.H{{"building_id": 203}, {"building_id": 203, "count": -1}, {"building_id": 201, "count": 1}, {"building_id": 999, "count": 1}, {"building_id": 203, "count": 1000000}, {"building_id": 203, "count": "bad"}} {
			rawAction(g, p, "set-auto-renew", action)
			if update := nextUpdate(t, g.Base.Players[uint64(p+1)].FailedUpdates); update.Kind != "set-auto-renew-failed" {
				t.Fatalf("invalid auto-renew did not report failure: %+v", update)
			}
		}
	})

	g.EndTurn()
	g.EndTurn()
	if !g.Self(p).ResearchedTechs[303] {
		t.Fatal("setup research did not complete")
	}
	g.SubmitRejected(p, research, "Tech id 303 is already researched")

	unchanged(g, "unknown or ended actions", func() {
		g.Risq.PlayerAction(game.PlayerAction{Client_id: 999999, Kind: "submit-orders"})
		rawAction(g, p, "unknown-action", nil)
		g.Base.EndGame("test")
		for kind, payload := range blockedActions(g, p, mine) {
			rawAction(g, p, kind, payload)
		}
		rawAction(g, p, "unsubmit-orders", nil)
	})
	if len(g.Failed(p)) != 0 {
		t.Fatal("ended game reported failures")
	}
}

// Each config restriction gets its own game so none can mask another
func TestOrderConfigRestrictions(t *testing.T) {
	origUnit, origTech := defs.UnitConfigs[11], defs.TechConfigs[301]
	t.Cleanup(func() { defs.UnitConfigs[11], defs.TechConfigs[301] = origUnit, origTech })
	for _, kind := range []string{"unit tech", "research tech", "build list", "unarmed unit"} {
		defs.UnitConfigs[11], defs.TechConfigs[301] = origUnit, origTech
		g := newGame(t, layout{tweak: func() {
			unit, tech := defs.UnitConfigs[11], defs.TechConfigs[301]
			switch kind {
			case "unit tech":
				unit.Required_tech_id = 304
			case "research tech":
				tech.Required_tech_id = 304
			case "build list":
				unit.Builds = nil
			case "unarmed unit":
				unit.Attack_type = defs.AttackType_NONE
			}
			defs.UnitConfigs[11], defs.TechConfigs[301] = unit, tech
		}})
		p := g.Human(0)
		mine, foe := scan(g, p), scan(g, g.Human(1))
		id := mine.workers[0]
		switch kind {
		case "unit tech":
			g.SubmitRejected(p, harness.OrderProduce([]uint64{mine.buildings[201]}, 11), "Required tech id 304 not researched for unit id 11")
		case "research tech":
			g.SubmitRejected(p, harness.Order(defs.OrderType_BuildingResearch, []uint64{mine.buildings[201]}, 301, false), "Required tech id 304 not researched for tech id 301")
		case "build list":
			g.SubmitRejected(p, harness.OrderBuild([]uint64{id}, 202, 1, 0, 0, 0), "Unit id 11 cannot build building id 202")
		case "unarmed unit":
			g.SubmitRejected(p, harness.OrderAttackUnit([]uint64{id}, foe.workers[0]), fmt.Sprintf("Unit id %d cannot attack", id))
		}
	}
}

func TestActionsRequireGivingOrders(t *testing.T) {
	g := newGame(t, layout{unstarted: true})
	p := g.Human(0)
	mine := scan(g, p)
	payloads := blockedActions(g, p, mine)
	payloads["unsubmit-orders"] = nil
	unchanged(g, "actions outside a turn", func() {
		for kind, payload := range payloads {
			rejectAction(g, p, kind, payload, "Not currently giving orders")
		}
	})
}
