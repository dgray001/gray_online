package shape

import "testing"

func TestTurnReportWireShape(t *testing.T) {
	g := shapeGame(t, 3)
	owner := g.Human(1)
	equal(t, player(t, snapshot(g, owner, false), 1)["turn_report"], nil)
	queueWork(g)
	p := player(t, snapshot(g, owner, false), 1)
	report := checkShape(t, p["turn_report"], "turn:n eliminated:b scores:a land:o regions:a resources:a population:o production:o combat:a orders:o")
	checkShape(t, report["land"], "held_start:n held_end:n gained:n lost:n newly_explored:n gold_from_land:n")
	checkShape(t, report["population"], "start:n end:n cap_start:n cap_end:n")
	production := checkShape(t, report["production"], "units_created:a buildings_built:a techs_researched:a")
	created := array(t, production["units_created"])
	equal(t, len(created), 1)
	checkShape(t, created[0], "unit_id:n count:n")
	orders := checkShape(t, report["orders"], "active:n added:n failed:n executed:n cancelled:n failures:a")
	equal(t, len(array(t, orders["failures"])), 0)
	resources := array(t, report["resources"])
	equal(t, len(resources), 4)
	for i, value := range resources {
		line := checkShape(t, value, "category:n start:n gathered:n spent:n final:n")
		equal(t, line["category"], float64(i+1))
	}
	for _, value := range array(t, report["scores"]) {
		checkShape(t, value, "player_id:n was:n now:n")
	}
	for _, value := range array(t, report["regions"]) {
		checkShape(t, value, "name:s held_start:b held_end:b gold_bonus:n")
	}
	equal(t, len(array(t, production["buildings_built"])), 0)
	equal(t, len(array(t, production["techs_researched"])), 0)
	equal(t, len(array(t, report["combat"])), 0)
}
