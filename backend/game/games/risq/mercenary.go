package risq

import (
	"maps"
	"slices"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/util"
	"github.com/gin-gonic/gin"
)

const mercenaryCostMarkup = 1.3
const mercenaryGoldMarkup = 1.5

func mercenaryCost(unit_id uint32) defs.RisqResourceCost {
	cost, _ := defs.UnitProductionCost(unit_id)
	total := cost.Food + cost.Wood + cost.Stone + cost.Gold*mercenaryGoldMarkup
	return defs.RisqResourceCost{Gold: total * mercenaryCostMarkup}
}

// Sorted by unit id so the frontend's button grid order is stable across updates
func (p *RisqPlayer) availableMercenariesToFrontend() []gin.H {
	mercenaries := make([]gin.H, 0, len(p.available_mercenaries))
	for _, unit_id := range slices.Sorted(maps.Keys(p.available_mercenaries)) {
		entry := defs.Producible{Kind: defs.ProducibleKind_UNIT, Id: unit_id}.ToFrontend()
		entry["cost"] = mercenaryCost(unit_id).ToFrontend()
		entry["stamina_cost"] = 0
		mercenaries = append(mercenaries, entry)
	}
	return mercenaries
}

func invertMercenaryKey(k uint, r *GameRisq) (uint32, *RisqSpace, *RisqZone) {
	unit_id, zone_key := util.InvertPair(k)
	space, zone := invertZoneKey(uint(zone_key), r)
	return uint32(unit_id), space, zone
}

type RisqPendingMercenary struct {
	unit_id   uint32
	zone      *RisqZone
	cost      defs.RisqResourceCost
	target_id int64
}

func (r *GameRisq) resolvePendingMercenaries() {
	for _, player := range r.players {
		if player.eliminated {
			continue
		}
		for _, pending := range player.pending_mercenaries {
			zone := pending.zone
			if zone.space.ownership != player.player.Player_id || zone.ownership != player.player.Player_id {
				player.resources.refund(pending.cost)
				player.report.recordFailure(defs.OrderType_BuyMercenary, pending.target_id, "zone lost ownership before placement")
				continue
			}
			if uint16(nonDeletedUnitCount(player.units)) >= player.populationLimit() {
				player.resources.refund(pending.cost)
				player.report.recordFailure(defs.OrderType_BuyMercenary, pending.target_id, "population capped before placement")
				continue
			}
			unit := createRisqUnit(r.nextUnitInternalId(), pending.unit_id, player)
			zone.space.setUnit(&zone.coordinate, unit)
			player.units[unit.internal_id] = unit
			r.units[unit.internal_id] = unit
			player.report.recordUnitCreated(pending.unit_id)
		}
		player.pending_mercenaries = player.pending_mercenaries[:0]
	}
}
