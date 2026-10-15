package risq

import (
	"encoding/json"
	"maps"
	"slices"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/gin-gonic/gin"
)

func (r *GameRisq) autoRenewWorkers(b *RisqBuilding) []*RisqUnit {
	workers := make([]*RisqUnit, 0)
	p := r.players[b.player_id]
	for _, id := range slices.Sorted(maps.Keys(p.units)) {
		u := p.units[id]
		orders := u.activeOrders()
		if !u.deleted && u.zone == b.zone && len(orders) == 1 && orders[0].order_type == defs.OrderType_UnitGather && orders[0].target_id == int64(b.zone.coordinate_key) {
			workers = append(workers, u)
		}
	}
	return workers
}

func (r *GameRisq) autoRenewBuildings() {
	for _, id := range slices.Sorted(maps.Keys(r.buildings)) {
		b := r.buildings[id]
		p := r.players[b.player_id]
		queue := p.auto_renewals[b.building_id]
		if b.deleted || b.underConstruction() || b.renewing != nil || b.resources_left > 0 || queue == nil || p.eliminated {
			continue
		}
		workers := r.autoRenewWorkers(b)
		if len(workers) == 0 {
			continue
		}
		cost := queue.costs[0]
		queue.costs = queue.costs[1:]
		b.startRenew(cost)
		if len(queue.costs) == 0 {
			delete(p.auto_renewals, b.building_id)
		}
		r.assignAutoRenewWorkers(b, workers)
	}
}

func (r *GameRisq) assignAutoRenewWorkers(b *RisqBuilding, workers []*RisqUnit) {
	for _, u := range workers {
		u.order_queue.nextOrder(u, r)
		order := createRisqOrder(r.nextOrderInternalId(), defs.OrderType_UnitRenew, b.player_id, map[uint64]Orderable{u.internal_id: u}, int64(b.internal_id), false)
		r.addSyntheticOrder(order, r.players[b.player_id], false)
	}
}

type AutoRenewFromFrontend struct {
	Building_id uint32 `json:"building_id"`
	Count       *int   `json:"count"`
}

func getAutoRenewFromPlayerAction(action gin.H) (AutoRenewFromFrontend, error) {
	var request AutoRenewFromFrontend
	data, err := json.Marshal(action)
	if err == nil {
		err = json.Unmarshal(data, &request)
	}
	return request, err
}

func (r *GameRisq) executeSetAutoRenew(player_id int, action gin.H) {
	p := r.players[player_id]
	request, err := getAutoRenewFromPlayerAction(action)
	if err != nil {
		p.player.AddFailedUpdateShorthand("set-auto-renew-failed", err.Error())
		return
	}
	if request.Count == nil {
		p.player.AddFailedUpdateShorthand("set-auto-renew-failed", "auto-renew count is required")
		return
	}
	if err := p.setAutoRenewCount(request.Building_id, *request.Count); err != nil {
		p.player.AddFailedUpdateShorthand("set-auto-renew-failed", err.Error())
		return
	}
	content := gin.H{"building_id": request.Building_id, "count": *request.Count, "resources": p.resources.toFrontend()}
	r.sendSetting(p.player, "auto-renew-set", content)
}
