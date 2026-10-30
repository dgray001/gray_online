package combat

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	testconfig "github.com/dgray001/gray_online/game/games/risq/tests/config"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
	"github.com/gin-gonic/gin"
)

func combatGame(t *testing.T, unit1, unit2 uint32) *harness.Game {
	t.Helper()
	return combatGameTweaked(t, unit1, unit2, func() {})
}

// Runs tweak after the config is loaded and before the units are created, so it changes their stats
func combatGameTweaked(t *testing.T, unit1, unit2 uint32, tweak func()) *harness.Game {
	t.Helper()
	u1 := ""
	if unit1 != 0 {
		u1 = fmt.Sprintf(`{"id":%d,"player":0,"count":1}`, unit1)
	}
	u2 := ""
	if unit2 != 0 {
		u2 = fmt.Sprintf(`{"id":%d,"player":1,"count":1}`, unit2)
	}
	units := ""
	if u1 != "" && u2 != "" {
		units = u1 + "," + u2
	} else {
		units = u1 + u2
	}
	doc := fmt.Sprintf(`{"board_size":1,"players":2,"spaces":[{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[%s]}]}]}`, units)
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"combat": doc})
	tweak()
	return harness.NewGame(t, "custom:combat", 1, 2)
}

func garrisonCombatGame(t *testing.T, race ...bool) *harness.Game {
	t.Helper()
	return garrisonCombatGameTweaked(t, func() {}, race...)
}

func garrisonCombatGameTweaked(t *testing.T, tweak func(), race ...bool) *harness.Game {
	t.Helper()
	// p0 gets an watchtower (id 21) and a phalanx (id 13). p1 gets a unit 13.
	count := 1
	if len(race) > 0 && race[0] {
		count = 2
	}
	doc := fmt.Sprintf(`{"board_size":1,"players":2,"spaces":[{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":207,"player":0},"units":[{"id":22,"player":0,"count":%d},{"id":22,"player":1,"count":1}]}]}]}`, count)
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"combat": doc})
	if count == 2 {
		config := defs.BuildingConfigs[207]
		config.Garrison_capacity = 1
		defs.BuildingConfigs[207] = config
	}
	tweak()
	return harness.NewGame(t, "custom:combat", 1, 2)
}

func combatGameTwoAttackers(t *testing.T) *harness.Game {
	t.Helper()
	doc := `{"board_size":1,"players":2,"spaces":[{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":21,"player":0,"count":2},{"id":22,"player":1,"count":1}]}]}]}`
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"combat": doc})
	return harness.NewGame(t, "custom:combat", 1, 2)
}

func retargetGame(t *testing.T) *harness.Game {
	t.Helper()
	doc := fmt.Sprintf(meetingBoard, spaceOf(0, 0, flat, placed{blunt, 0, 1, 0}, placed{11, 1, 1, 0}, placed{11, 1, 1, 0}))
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"retarget": doc})
	config := defs.UnitConfigs[blunt]
	config.Attack_blunt = 10
	defs.UnitConfigs[blunt] = config
	return harness.NewGame(t, "custom:retarget", 1, 2)
}

func checkOutcome(t *testing.T, snapshot gin.H, winners []int) {
	t.Helper()
	outcome, ok := snapshot["outcome"].(gin.H)
	if !ok || !reflect.DeepEqual(outcome["winner_player_ids"], winners) {
		t.Fatalf("outcome = %v, want winners %v", snapshot["outcome"], winners)
	}
	if snapshot["game_base"].(gin.H)["game_ended"] != true || snapshot["giving_orders"] != false {
		t.Fatalf("final snapshot is still running: %v", snapshot)
	}
	for _, player := range snapshot["players"].([]gin.H) {
		if len(player["active_orders"].([]gin.H)) != 0 {
			t.Fatal("final snapshot contains player orders")
		}
		for _, field := range []string{"units", "buildings"} {
			for _, entity := range player[field].([]gin.H) {
				if len(entity["active_orders"].([]gin.H)) != 0 {
					t.Fatal("final snapshot contains entity orders")
				}
			}
		}
	}
}

var testConfig = testconfig.Dir()

func TestMain(m *testing.M) {
	if err := defs.LoadConfig(testConfig); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}
