package combat

import (
	"encoding/json"
	"fmt"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
	"testing"
)

func movementCost(t *testing.T, dir string, cost uint) {
	t.Helper()
	data, err := defs.ReadConfigFile("terrains.json")
	if err != nil {
		t.Fatal(err)
	}
	var entries []map[string]any
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry["terrain_type"] == "flatlands" {
			entry["intra_cost"] = cost
		}
	}
	data, err = json.Marshal(entries)
	if err != nil {
		t.Fatal(err)
	}
	if err := defs.WriteConfigFile(data, "terrains.json"); err != nil {
		t.Fatal(err)
	}
	if err := defs.LoadConfig(dir); err != nil {
		t.Fatal(err)
	}
}

func meetingBoundaryGame(t *testing.T, cost uint, stamina, targetStamina int) *harness.Game {
	t.Helper()
	doc := fmt.Sprintf(meetingBoard, spaceOf(0, 0, flat, placed{blunt, 0, 0, 0}, placed{heavy, 1, 1, 0}))
	dir := fakeboard.UseConfig(t, testConfig, nil, map[string]string{"meeting-boundary": doc})
	movementCost(t, dir, cost)
	for id, grant := range map[uint32]int{blunt: stamina, heavy: targetStamina} {
		config := defs.UnitConfigs[id]
		config.Turn_stamina = grant
		defs.UnitConfigs[id] = config
	}
	return harness.NewGame(t, "custom:meeting-boundary", 1, 2)
}
