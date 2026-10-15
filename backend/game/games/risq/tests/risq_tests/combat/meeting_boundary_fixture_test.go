package combat

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
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

func TestMeetingStaminaBoundaries(t *testing.T) {
	for _, c := range []struct {
		cost                  uint
		stamina, sunk, attack int
	}{
		{1, 15, 1, 2}, {2, 15, 1, 2}, {3, 15, 2, 1}, {4, 15, 2, 1}, {5, 15, 3, 0}, {6, 15, 3, 0}, {9, 15, 5, 0}, {12, 15, 6, 0}, {1, 2, 1, 1}, {3, 3, 2, 1}, {1, 1, 1, 0},
	} {
		t.Run(fmt.Sprintf("cost-%d-stamina-%d", c.cost, c.stamina), func(t *testing.T) {
			log := captureTicks(t)
			g := meetingBoundaryGame(t, c.cost, c.stamina, 15)
			a, b := soleUnit(g, 0), soleUnit(g, 1)
			makePassive(g, 1, b)
			g.Submit(g.Human(0), attackOrder(a, b))
			g.Submit(g.Human(1), harness.OrderMoveZone([]uint64{b.InternalID}, 0, 0, 0, 0))
			ticks := parseTicks(t, log)
			meeting, found := ticks.meetingOf(int(a.InternalID))
			if !found || meeting.tick != 1 || meeting.sunk != c.sunk || meeting.attack != c.attack {
				t.Fatalf("meeting=%+v found=%v, want sunk %d attack %d", meeting, found, c.sunk, c.attack)
			}
			hits := ticks.hitsBy(int(a.InternalID))
			firstTickAttack := len(hits) > 0 && hits[0].tick == 1
			if firstTickAttack != (c.attack > 0) {
				t.Fatalf("first tick hits=%+v", hits)
			}
			if firstTickAttack && hits[0].stamina != c.attack {
				t.Fatalf("attack stamina=%d, want %d", hits[0].stamina, c.attack)
			}
		})
	}
}
