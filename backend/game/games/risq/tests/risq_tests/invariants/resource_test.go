package invariants

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

func checkResource(t *testing.T, resource, space, zone object) {
	t.Helper()
	config := defs.ResourceConfigs[uint32(number(resource, "resource_id"))]
	bound(t, "node resources", number(resource, "resources_left"), 0, config.Starting_resources)
	if number(resource, "resources_left") == 0 {
		t.Error("depleted resource node was not removed")
	}
	if canonical(resource["space_coordinate"]) != canonical(space["coordinate"]) || canonical(resource["zone_coordinate"]) != canonical(zone["coordinate"]) {
		t.Error("resource coordinates disagree with zone")
	}
}
