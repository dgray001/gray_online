package ai

type gatherableItem struct {
	place
	node     ResourceView
	building *BuildingView
}

func (g gatherableItem) field(name string) (float64, bool) {
	return gatherableFields.get(g, name)
}

func (g gatherableItem) owner() int {
	if g.building == nil {
		return -1
	}
	return g.building.PlayerID
}

var gatherableFields = withPlace(fieldTable[gatherableItem]{
	"uid":             func(g gatherableItem) float64 { return float64(g.node.uid()) },
	"category":        func(g gatherableItem) float64 { return float64(g.node.Category) },
	"amount_left":     func(g gatherableItem) float64 { return g.node.AmountLeft },
	"gather_speed":    func(g gatherableItem) float64 { return float64(g.node.GatherSpeed) },
	"gather_capacity": func(g gatherableItem) float64 { return float64(g.node.GatherCapacity) },
	"is_building":     func(g gatherableItem) float64 { return boolNumber(g.building != nil) },
	"owner":           func(g gatherableItem) float64 { return float64(g.owner()) },
}, func(g gatherableItem) place { return g.place })

func gatherableAim(g gatherableItem) aim {
	if g.building != nil {
		return aim{building: g.building, loc: g.loc}
	}
	return aim{resource: &g.node, loc: g.loc}
}

type gatherableSource struct{ resources *resourceSource }

func parseGatherableSource(where map[string]any) (loopSource, error) {
	resources, err := newResourceSource(where)
	return &gatherableSource{resources: resources}, err
}

func (s *gatherableSource) items(view View, internals *Internals) []loopItem {
	items := make([]loopItem, 0)
	for _, node := range s.resources.nodes(view) {
		if node.AmountLeft > 0 {
			items = append(items, gatherableItem{place: place{loc: node.Location}, node: node})
		}
	}
	for _, b := range view.Buildings() {
		if b.Gatherable && !b.UnderConstruction && b.ResourcesLeft > 0 && s.resources.takes(b.ResourceCategory) {
			items = append(items, gatherableItem{place: place{loc: b.Location}, node: gatherableAsResource(b), building: &b})
		}
	}
	return items
}
