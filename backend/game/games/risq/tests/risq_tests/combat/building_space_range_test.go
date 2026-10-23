package combat

import (
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

const watchtower = 207

// A space-range watchtower in towerZone of space (0,0) beside a worker, against one passive phalanx; returns whether it was shot in one turn
func towerHurtsEnemy(t *testing.T, towerZone harness.Coord, enemy enemyPlacement) bool {
	t.Helper()
	log := captureTicks(t)
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"tower": fmt.Sprintf(meetingBoard, towerSpaces(towerZone, enemy))})
	config := defs.BuildingConfigs[watchtower]
	config.Turn_stamina = 10
	defs.BuildingConfigs[watchtower] = config
	g := harness.NewGame(t, "custom:tower", 1, 2)
	foe := g.Human(1)
	passive(g, foe, g.Self(foe).Units[0].InternalID)
	g.EndTurn()
	return len(parseTicks(t, log).hits) > 0
}

func towerSpaces(towerZone harness.Coord, enemy enemyPlacement) string {
	worker := fmt.Sprintf(zoneJSON, -1, 1, fmt.Sprintf(unitJSON, 11, 0))
	tower := fmt.Sprintf(`{"x":%d,"y":%d,"building":{"id":%d,"player":0}}`, towerZone.X, towerZone.Y, watchtower)
	phalanx := fmt.Sprintf(zoneJSON, enemy.zone.X, enemy.zone.Y, fmt.Sprintf(unitJSON, heavy, 1))
	if enemy.space == center {
		return fmt.Sprintf(spaceJSON, 0, 0, flat, worker+","+tower+","+phalanx)
	}
	home := fmt.Sprintf(spaceJSON, 0, 0, flat, worker+","+tower)
	return home + "," + fmt.Sprintf(spaceJSON, enemy.space.X, enemy.space.Y, flat, phalanx)
}

func TestSpaceRangeBuildingReach(t *testing.T) {
	for _, c := range []struct {
		name  string
		tower harness.Coord
		enemy enemyPlacement
		hurt  bool
	}{
		{"edge tower hits facing zone", east, eastFacing, true},
		{"edge tower hits own space", east, sameWest, true},
		{"edge tower ignores neighbor center", east, eastCenter, false},
		{"center tower ignores facing zone", center, eastFacing, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			if hurt := towerHurtsEnemy(t, c.tower, c.enemy); hurt != c.hurt {
				t.Errorf("hurt=%v, want %v", hurt, c.hurt)
			}
		})
	}
}
