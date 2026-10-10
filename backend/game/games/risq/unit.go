package risq

import (
	"fmt"
	"os"
	"slices"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/util"
	"github.com/gin-gonic/gin"
)

type RisqUnit struct {
	orderableBase
	unit_id       uint32
	display_name  string
	garrisoned_in *RisqBuilding
	stance        defs.UnitStance
	attack_back   bool
	// the source this unit gathered from last tick; holders keep their gather slot over newcomers
	gather_slot Gatherable
	// this tick's full planned route, for frontend display only; cleared whenever the unit isn't actively moving
	move_path []*RisqZone
	// the half move already paid toward an adjacent zone; kept only while each tick's intent keeps stepping there
	half_move *halfMove
}

func createRisqUnit(internal_id uint64, unit_id uint32, player *RisqPlayer) *RisqUnit {
	unit := RisqUnit{
		orderableBase: orderableBase{
			internal_id:     internal_id,
			player_id:       player.player.Player_id,
			turn_stamina:    10,
			cs:              createRisqCombatStats(),
			order_queue:     createRisqOrderQueue(),
			intent:          createRisqIntent(),
			target_priority: []defs.TargetCategory{},
		},
		unit_id:     unit_id,
		stance:      defs.UnitStance_PASSIVE,
		attack_back: true,
	}
	config, ok := defs.UnitConfigs[unit_id]
	if !ok {
		fmt.Fprintln(os.Stderr, "Creating unknown unit id: ", unit_id)
		return &unit
	}
	if config.Unit_type != defs.UnitType_ECONOMIC {
		unit.stance = player.default_unit_stance
		unit.attack_back = player.default_unit_attack_back
		unit.interrupt_current = player.default_unit_interrupt_current
		unit.target_priority = slices.Clone(player.default_unit_target_priority)
	}
	unit.display_name = config.Display_name
	unit.cs.setMaxHealth(config.Max_health)
	unit.cs.attack_type = config.Attack_type
	unit.cs.attack_blunt = config.Attack_blunt
	unit.cs.attack_piercing = config.Attack_piercing
	unit.cs.defense_blunt = config.Defense_blunt
	unit.cs.defense_piercing = config.Defense_piercing
	unit.cs.penetration_blunt = config.Penetration_blunt
	unit.cs.penetration_piercing = config.Penetration_piercing
	unit.attack_range = config.Attack_range
	unit.turn_stamina = config.Turn_stamina
	for _, bonus := range defs.BonusConfigs {
		if bonus.AppliesTo(unit_id, config.Unit_type) {
			applyTechBonus(&unit, bonus)
		}
	}
	for tech_id, researched := range player.researched_techs {
		if !researched {
			continue
		}
		tech, ok := defs.TechConfigs[tech_id]
		if !ok || !tech.AppliesTo(unit_id, config.Unit_type) {
			continue
		}
		applyTechBonus(&unit, tech)
	}
	return &unit
}

func (u *RisqUnit) vision() *defs.RisqVision {
	v := defs.UnitConfigs[u.unit_id].Vision
	return &v
}

func (u *RisqUnit) score() uint {
	return defs.UnitConfigs[u.unit_id].Cost.Points()
}

func (u *RisqUnit) unitType() defs.UnitType {
	return defs.UnitConfigs[u.unit_id].Unit_type
}

func (u *RisqUnit) OrderableType() defs.OrderableType {
	return defs.OrderableType_UNIT
}

func (u *RisqUnit) combatStats(r *GameRisq, other Orderable, attacking bool) RisqCombatStats {
	return r.effectiveCombatStats(u, other, attacking)
}

func (u *RisqUnit) resolveHealthDelta(r *GameRisq) {
	if u.cs.pending_health_delta == 0 {
		return
	}
	was_alive := u.isAlive()
	util.DebugLog.Printf("health turn=%d tick=%d: %d u%d before=%.4f delta=%.4f", r.turn_number, r.current_tick, u.internal_id, u.unit_id, u.cs.health, u.cs.pending_health_delta)
	u.cs.addHealth(u.cs.pending_health_delta)
	u.cs.pending_health_delta = 0
	if was_alive && !u.isAlive() {
		if event, ok := lowestIdAttackerEvent(u.attacked_by, r.current_tick); ok {
			if attacker := r.resolveAttacker(event); attacker != nil {
				u.recordDeath(r, attacker, event.damage)
			}
		}
	}
}

func (u *RisqUnit) recordDeath(r *GameRisq, attacker Attackable, damage float64) {
	death_zone := u.zone
	if death_zone == nil && u.garrisoned_in != nil {
		death_zone = u.garrisoned_in.zone
	}
	death_zone.space.death_vision[u.player_id] = true
	death_zone.corpses[u.internal_id] = RisqCorpse{unit_id: u.unit_id, player_id: u.player_id, turns: 1}
	space := death_zone.space.coordinate
	zone := death_zone.coordinate
	r.players[attacker.playerId()].report.recordCombat(RisqCombatEvent{tick: r.current_tick, kind: RisqCombatEventKind_UNIT_KILLED,
		self_player: attacker.playerId(), other_player: u.player_id, target_id: uint64(u.unit_id), space: space, zone: zone, damage: damage})
	r.players[u.player_id].report.recordCombat(RisqCombatEvent{tick: r.current_tick, kind: RisqCombatEventKind_UNIT_LOST,
		self_player: u.player_id, other_player: attacker.playerId(), target_id: uint64(u.unit_id), space: space, zone: zone, damage: damage})
	r.players[attacker.playerId()].kills++
	if !u.deleted {
		r.players[u.player_id].units_lost++
	}
	if u.unitType() == defs.UnitType_ECONOMIC {
		r.players[u.player_id].economy.villagers_lost++
	}
	u.deleted = true
}

func (u *RisqUnit) cleanupDeleted(risq *GameRisq) {
	resolveOrdersOnDeath(risq, &u.order_queue, u.internal_id, defs.OrderType_UnitDelete)
	if u.zone != nil && u.zone.space != nil {
		u.zone.space.removeUnit(u)
	}
	if u.garrisoned_in != nil {
		delete(u.garrisoned_in.garrisoned_units, u.internal_id)
		u.garrisoned_in = nil
	}
	delete(risq.players[u.player_id].units, u.internal_id)
	delete(risq.units, u.internal_id)
}

func (u *RisqUnit) toFrontend(viewer_player_id int) gin.H {
	unit := gin.H{
		"internal_id":     u.internal_id,
		"player_id":       u.player_id,
		"unit_id":         u.unit_id,
		"unit_type":       u.unitType(),
		"display_name":    u.display_name,
		"turn_stamina":    u.turn_stamina,
		"current_stamina": u.current_stamina,
		"max_stamina":     maxStaminaFor(u.turn_stamina),
		"combat_stats":    u.cs.toFrontend(),
		"attack_range":    u.attack_range,
	}
	if u.garrisoned_in != nil {
		unit["garrisoned_in"] = u.garrisoned_in.internal_id
	}
	builds := make([]gin.H, 0)
	for _, p := range defs.UnitConfigs[u.unit_id].Builds {
		builds = append(builds, p.ToFrontend())
	}
	unit["builds"] = builds
	if u.zone != nil {
		unit["zone_coordinate"] = u.zone.coordinate.ToFrontend()
		if u.zone.space != nil {
			unit["space_coordinate"] = u.zone.space.coordinate.ToFrontend()
		}
	}
	active_orders := make([]gin.H, 0)
	if showOrdersTo(u.player_id, u.zone, viewer_player_id) {
		for _, order := range u.order_queue.active_orders {
			if order != nil && !order.executed {
				active_orders = append(active_orders, order.toFrontend())
			}
		}
		target_priority := make([]int, len(u.target_priority))
		for i, cat := range u.target_priority {
			target_priority[i] = int(cat)
		}
		unit["stance"] = u.stance
		unit["interrupt_current"] = u.interrupt_current
		unit["attack_back"] = u.attack_back
		unit["target_priority"] = target_priority
		if len(u.move_path) > 0 {
			move_path := make([]gin.H, len(u.move_path))
			for i, zone := range u.move_path {
				move_path[i] = gin.H{"space": zone.space.coordinate.ToFrontend(), "zone": zone.coordinate.ToFrontend()}
			}
			unit["move_path"] = move_path
		}
	}
	unit["active_orders"] = active_orders
	unit["tick_actions"] = tickActionsToFrontend(u.tick_actions, viewer_player_id)
	return unit
}
