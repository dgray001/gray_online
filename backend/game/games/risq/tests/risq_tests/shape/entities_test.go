package shape

import "testing"

func TestOwnedEntityWireShapes(t *testing.T) {
	g := shapeGame(t, 3)
	owner := player(t, snapshot(g, g.Human(1), false), 1)
	for _, value := range array(t, owner["units"]) {
		unit := checkShape(t, value, unitShape+" "+privateUnitShape)
		checkEntityDetails(t, unit, "builds")
	}
	for _, value := range array(t, owner["buildings"]) {
		building := object(t, value)
		contract := buildingShape + " " + privateBuildingShape
		if building["building_id"] == float64(3) {
			contract += " " + farmShape
			checkShape(t, building["renew_cost"], costShape)
		}
		checkShape(t, building, contract)
		checkEntityDetails(t, building, "produces")
	}
	target := space(t, snapshot(g, g.Human(1), false), 0, 0)
	resources := array(t, target["resources"])
	equal(t, len(resources), 1)
	resource := checkShape(t, resources[0], resourceShape)
	checkShape(t, resource["space_coordinate"], coordinateShape)
	checkShape(t, resource["zone_coordinate"], coordinateShape)
	equal(t, resource["resource_id"], float64(41))
}
