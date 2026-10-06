package shape

import "testing"

func checkFogContents(t *testing.T, target map[string]any) {
	t.Helper()
	for _, value := range array(t, target["resources"]) {
		resource := checkShape(t, value, resourceShape)
		checkShape(t, resource["space_coordinate"], coordinateShape)
		checkShape(t, resource["zone_coordinate"], coordinateShape)
	}
	for _, row := range array(t, target["zones"]) {
		for _, value := range array(t, row) {
			zone := object(t, value)
			checkShape(t, zone, targetZoneContract(t, zone))
			if zone["resource"] != nil {
				checkShape(t, zone["resource"], resourceShape)
			}
			if zone["building"] != nil {
				checkShape(t, zone["building"], cachedBuildingShape)
			}
		}
	}
}
