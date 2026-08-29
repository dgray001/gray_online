package risq

const productionTickStaminaCost = 5

type ProductionIntent struct {
	item *RisqBuildingProductionItem
}

func (*ProductionIntent) isIntentKind() {}

func (i *RisqIntent) setProduce(item *RisqBuildingProductionItem) {
	i.detail = &ProductionIntent{item: item}
	i.min_cost = 1
	i.max_cost = productionTickStaminaCost
}
