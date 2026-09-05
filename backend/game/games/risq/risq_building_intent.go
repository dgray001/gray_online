package risq

const buildingTickStaminaCost = 5

type ProductionIntent struct {
	order_internal_id uint64
	item              *RisqBuildingProductionItem
}

func (*ProductionIntent) isIntentKind() {}

func (i *RisqIntent) setProduction(order_internal_id uint64, item *RisqBuildingProductionItem) {
	i.detail = &ProductionIntent{order_internal_id: order_internal_id, item: item}
	i.min_cost = 1
	i.max_cost = buildingTickStaminaCost
}
