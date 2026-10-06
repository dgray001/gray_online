package ai

import (
	"slices"
	"testing"
)

type workforceTestView struct {
	View
	units []UnitView
}

func (v workforceTestView) EligibleUnits(kinds ...OrderKind) []UnitView {
	units := make([]UnitView, 0)
	for _, unit := range v.units {
		if unit.CurrentOrder == nil || slices.Contains(kinds, unit.CurrentOrder.Kind) {
			units = append(units, unit)
		}
	}
	return units
}

func TestAvailableGatherers(t *testing.T) {
	garrison := uint64(1)
	view := workforceTestView{units: []UnitView{
		{Kind: UnitEconomic},
		{Kind: UnitEconomic, CurrentOrder: &CurrentOrder{Kind: OrderKindGather}},
		{Kind: UnitEconomic, GarrisonedIn: &garrison},
		{Kind: UnitMilitary},
	}}
	for _, kind := range []OrderKind{OrderKindBuild, OrderKindRepair, OrderKindRenew, OrderKindMove, OrderKindAttackSpace, OrderKindGarrison} {
		view.units = append(view.units, UnitView{Kind: UnitEconomic, CurrentOrder: &CurrentOrder{Kind: kind}})
	}
	if got := availableGatherers(view, nil); got != 2 {
		t.Fatalf("available gatherers = %v, want idle villager and gatherer only", got)
	}
}
