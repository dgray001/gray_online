package risq

import (
	"errors"
	"maps"
	"slices"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/gin-gonic/gin"
)

type RisqRenewalQueue struct {
	costs []defs.RisqResourceCost
}

func (p *RisqPlayer) setAutoRenewCount(id uint32, count int) error {
	config, ok := defs.BuildingConfigs[id]
	if !ok || !config.IsGatherable() {
		return errors.New("building type is not gatherable")
	}
	if count < 0 {
		return errors.New("auto-renew count cannot be negative")
	}
	queue := p.auto_renewals[id]
	if queue == nil {
		queue = &RisqRenewalQueue{}
	}
	cost := config.Gather.Renew_cost
	additional := count - len(queue.costs)
	if additional > 0 && !p.resources.canAfford(cost.Scale(float64(additional))) {
		return errors.New("cannot afford auto-renew")
	}
	p.resizeAutoRenewQueue(id, queue, count, cost)
	return nil
}

func (p *RisqPlayer) resizeAutoRenewQueue(id uint32, queue *RisqRenewalQueue, count int, cost defs.RisqResourceCost) {
	for len(queue.costs) < count {
		p.resources.spend(cost)
		queue.costs = append(queue.costs, cost)
	}
	for len(queue.costs) > count {
		last := len(queue.costs) - 1
		p.resources.refund(queue.costs[last])
		queue.costs = queue.costs[:last]
	}
	if count == 0 {
		delete(p.auto_renewals, id)
	} else {
		p.auto_renewals[id] = queue
	}
}

func (p *RisqPlayer) autoRenewalsToFrontend() []gin.H {
	queues := make([]gin.H, 0, len(p.auto_renewals))
	for _, id := range slices.Sorted(maps.Keys(p.auto_renewals)) {
		queues = append(queues, gin.H{"building_id": id, "count": len(p.auto_renewals[id].costs)})
	}
	return queues
}
