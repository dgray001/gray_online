package risq

import (
	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/gin-gonic/gin"
)

type RisqTurnReport struct {
	turn             uint16
	score_start      uint
	explored_start   int
	scores           []RisqScoreLine
	land             RisqLandReport
	resources        [4]RisqResourceLine
	pop_start        uint16
	pop_end          uint16
	cap_start        uint16
	cap_end          uint16
	units_created    map[uint32]int
	buildings_built  []RisqBuiltEntry
	techs_researched []uint32
	combat           []RisqCombatEvent
	orders           RisqOrdersReport
}

type RisqScoreLine struct {
	player_id int
	was       uint
	now       uint
}

type RisqLandReport struct {
	held_start     int
	held_end       int
	gained         int
	lost           int
	newly_explored int
	gold_from_land float64
	start_keys     map[uint]bool
}

type RisqResourceLine struct {
	start    float64
	gathered float64
	spent    float64
}

type RisqBuiltEntry struct {
	building_id uint32
	space       game_utils.Coordinate2D
	zone        game_utils.Coordinate2D
}

type RisqOrdersReport struct {
	active    int
	added     int
	failed    int
	executed  int
	cancelled int
	failures  []RisqOrderFailure
}

type RisqOrderFailure struct {
	order_type OrderType
	target_id  int64
	reason     string
}

type RisqCombatEventKind uint8

const (
	CombatEvent_BuildingRazed RisqCombatEventKind = iota
	CombatEvent_BuildingLost
	CombatEvent_UnitKilled
	CombatEvent_UnitLost
)

type RisqCombatEvent struct {
	tick         uint16
	kind         RisqCombatEventKind
	self_player  int
	other_player int
	target_id    uint64
	space        game_utils.Coordinate2D
	zone         game_utils.Coordinate2D
	damage       float64
}

func createRisqTurnReport(turn uint16, p *RisqPlayer, owned map[uint]bool) *RisqTurnReport {
	rep := &RisqTurnReport{
		turn:             turn,
		pop_start:        uint16(len(p.units)),
		cap_start:        p.populationLimit(),
		units_created:    make(map[uint32]int),
		techs_researched: make([]uint32, 0),
	}
	rep.land.start_keys = owned
	rep.land.held_start = len(owned)
	res := p.resources
	rep.resources[RisqResourceCategory_FOOD].start = res.food
	rep.resources[RisqResourceCategory_WOOD].start = res.wood
	rep.resources[RisqResourceCategory_STONE].start = res.stone
	rep.resources[RisqResourceCategory_GOLD].start = res.gold
	return rep
}

func (r *RisqTurnReport) recordUnitCreated(unit_id uint32) {
	if r == nil {
		return
	}
	r.units_created[unit_id]++
}

func (r *RisqTurnReport) recordBuildingBuilt(building_id uint32, space game_utils.Coordinate2D, zone game_utils.Coordinate2D) {
	if r == nil {
		return
	}
	r.buildings_built = append(r.buildings_built, RisqBuiltEntry{building_id: building_id, space: space, zone: zone})
}

func (r *RisqTurnReport) recordTech(tech_id uint32) {
	if r == nil {
		return
	}
	r.techs_researched = append(r.techs_researched, tech_id)
}

func (r *RisqTurnReport) recordGoldFromLand(amount float64) {
	if r == nil {
		return
	}
	r.land.gold_from_land += amount
}

func (r *RisqTurnReport) recordCombat(e RisqCombatEvent) {
	if r == nil {
		return
	}
	r.combat = append(r.combat, e)
}

func (r *RisqTurnReport) recordFailure(order_type OrderType, target_id int64, reason string) {
	if r == nil {
		return
	}
	r.orders.failed++
	r.orders.failures = append(r.orders.failures, RisqOrderFailure{order_type: order_type, target_id: target_id, reason: reason})
}

func (r *GameRisq) ownedSpaceKeys(player_id int) map[uint]bool {
	owned := make(map[uint]bool)
	for _, row := range r.spaces {
		for _, s := range row {
			if s.ownership == player_id {
				owned[s.coordinate_key] = true
			}
		}
	}
	return owned
}

func (r *GameRisq) exploredCount(player_id int) int {
	count := 0
	for _, row := range r.spaces {
		for _, s := range row {
			if s.getVisibility(player_id) >= VisibilityFog {
				count++
			}
		}
	}
	return count
}

func (r *GameRisq) ownedCount(player_id int) int {
	count := 0
	for _, row := range r.spaces {
		for _, s := range row {
			if s.ownership == player_id {
				count++
			}
		}
	}
	return count
}

func (r *GameRisq) computePlayerScore(p *RisqPlayer) uint {
	score := p.resources.score()
	for _, u := range p.units {
		if u != nil && !u.deleted {
			score += u.score()
		}
	}
	for _, b := range p.buildings {
		if b != nil && !b.deleted {
			score += b.score()
		}
	}
	for tech_id, done := range p.researched_techs {
		if done {
			score += techConfigs[tech_id].cost.points()
		}
	}
	pid := p.player.Player_id
	score += uint(r.exploredCount(pid)) * 10
	score += uint(r.ownedCount(pid)) * 100
	return score
}

func (r *GameRisq) refreshScores() {
	for _, p := range r.players {
		p.score = r.computePlayerScore(p)
	}
}

func (r *GameRisq) beginTurnReports() {
	for _, p := range r.players {
		p.report = createRisqTurnReport(r.turn_number, p, r.ownedSpaceKeys(p.player.Player_id))
		p.report.score_start = p.score
		p.report.explored_start = r.exploredCount(p.player.Player_id)
		p.resources.resetFlow()
	}
}

func (r *GameRisq) finalizeTurnReports() {
	scores := make([]RisqScoreLine, 0, len(r.players))
	for _, p := range r.players {
		if p.report == nil {
			continue
		}
		scores = append(scores, RisqScoreLine{player_id: p.player.Player_id, was: p.report.score_start, now: p.score})
	}
	for _, p := range r.players {
		rep := p.report
		if rep == nil {
			continue
		}
		pid := p.player.Player_id
		end := r.ownedSpaceKeys(pid)
		rep.land.held_end = len(end)
		for key := range end {
			if !rep.land.start_keys[key] {
				rep.land.gained++
			}
		}
		for key := range rep.land.start_keys {
			if !end[key] {
				rep.land.lost++
			}
		}
		rep.land.newly_explored = r.exploredCount(pid) - rep.explored_start
		for cat := 0; cat < 4; cat++ {
			rep.resources[cat].gathered = p.resources.gathered[cat]
			rep.resources[cat].spent = p.resources.spent[cat]
		}
		rep.pop_end = uint16(len(p.units))
		rep.cap_end = p.populationLimit()
		rep.scores = scores
	}
}

func (e RisqCombatEvent) toFrontend() gin.H {
	return gin.H{
		"tick":         e.tick,
		"kind":         e.kind,
		"self_player":  e.self_player,
		"other_player": e.other_player,
		"target_id":    e.target_id,
		"space":        e.space.ToFrontend(),
		"zone":         e.zone.ToFrontend(),
		"damage":       e.damage,
	}
}

func (rep *RisqTurnReport) toFrontend() gin.H {
	if rep == nil {
		return nil
	}
	scores := make([]gin.H, 0, len(rep.scores))
	for _, s := range rep.scores {
		scores = append(scores, gin.H{"player_id": s.player_id, "was": s.was, "now": s.now})
	}
	resources := make([]gin.H, 0, 4)
	for cat := 0; cat < 4; cat++ {
		line := rep.resources[cat]
		resources = append(resources, gin.H{
			"category": cat,
			"start":    line.start,
			"gathered": line.gathered,
			"spent":    line.spent,
			"final":    line.start + line.gathered - line.spent,
		})
	}
	units := make([]gin.H, 0, len(rep.units_created))
	for uid, count := range rep.units_created {
		units = append(units, gin.H{"unit_id": uid, "count": count})
	}
	buildings := make([]gin.H, 0, len(rep.buildings_built))
	for _, b := range rep.buildings_built {
		buildings = append(buildings, gin.H{"building_id": b.building_id, "space": b.space.ToFrontend(), "zone": b.zone.ToFrontend()})
	}
	combat := make([]gin.H, 0, len(rep.combat))
	for _, e := range rep.combat {
		combat = append(combat, e.toFrontend())
	}
	failures := make([]gin.H, 0, len(rep.orders.failures))
	for _, f := range rep.orders.failures {
		failures = append(failures, gin.H{"order_type": f.order_type, "target_id": f.target_id, "reason": f.reason})
	}
	return gin.H{
		"turn":   rep.turn,
		"scores": scores,
		"land": gin.H{
			"held_start":     rep.land.held_start,
			"held_end":       rep.land.held_end,
			"gained":         rep.land.gained,
			"lost":           rep.land.lost,
			"newly_explored": rep.land.newly_explored,
			"gold_from_land": rep.land.gold_from_land,
		},
		"resources":  resources,
		"population":  gin.H{"start": rep.pop_start, "end": rep.pop_end, "cap_start": rep.cap_start, "cap_end": rep.cap_end},
		"production":  gin.H{"units_created": units, "buildings_built": buildings, "techs_researched": rep.techs_researched},
		"combat":      combat,
		"orders": gin.H{
			"active":    rep.orders.active,
			"added":     rep.orders.added,
			"failed":    rep.orders.failed,
			"executed":  rep.orders.executed,
			"cancelled": rep.orders.cancelled,
			"failures":  failures,
		},
	}
}
