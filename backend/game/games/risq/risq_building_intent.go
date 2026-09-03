package risq

const productionTickStaminaCost = 5

type CreateIntent struct {
	order_internal_id uint64
	item              *RisqBuildingProductionItem
}

func (*CreateIntent) isIntentKind() {}

func (i *RisqIntent) setCreate(order_internal_id uint64, item *RisqBuildingProductionItem) {
	i.detail = &CreateIntent{order_internal_id: order_internal_id, item: item}
	i.min_cost = 1
	i.max_cost = productionTickStaminaCost
}
