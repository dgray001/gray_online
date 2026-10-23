package ai

type foundationItem struct {
	place
	foundation FoundationView
}

func (f foundationItem) field(name string) (float64, bool) {
	return foundationFields.get(f, name)
}

type foundationSource struct {
	ids buildingFilter
}

func parseFoundationSource(where map[string]any) (loopSource, error) {
	if err := checkKeys(where, "building_ids"); err != nil {
		return nil, err
	}
	ids, err := parseIDSet(where, "building_ids")
	return &foundationSource{ids: buildingFilter{ids: ids}}, err
}

func (s *foundationSource) items(view View, _ *Internals) []loopItem {
	items := make([]loopItem, 0)
	for _, f := range view.Foundations() {
		if s.ids.matches(f.BuildingID) {
			items = append(items, foundationItem{place: place{loc: f.Location}, foundation: f})
		}
	}
	return items
}

func (a *selectAction) cancelFoundations(view View, internals *Internals, targets []selItem) []Order {
	orders := make([]Order, 0, len(targets))
	for _, target := range targets {
		aimed, ok := aimAt(target)
		if !ok || aimed.foundation == nil || !aimed.foundation.Planned || aimed.foundation.OrderedThisTurn {
			continue
		}
		cost := view.BuildCost(aimed.foundation.BuildingID)
		internals.spend(Cost{Food: -cost.Food, Wood: -cost.Wood, Stone: -cost.Stone, Gold: -cost.Gold})
		orders = append(orders, view.Issue(IssueRequest{Verb: VerbCancelFoundation, Foundation: *aimed.foundation}))
	}
	return orders
}
