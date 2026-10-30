package combat

import (
	"fmt"
	"strings"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

const watchtower = 207

// A space-range watchtower in towerZone of space (0,0) beside a worker, against one passive phalanx; returns whether it was shot in one turn
func towerHurtsEnemy(t *testing.T, towerZone harness.Coord, enemy enemyPlacement, ranges ...defs.RisqRange) bool {
	t.Helper()
	log := captureTicks(t)
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"tower": fmt.Sprintf(`{"board_size":3,"players":2,"spaces":[%s]}`, towerSpaces(towerZone, enemy))})
	config := defs.BuildingConfigs[watchtower]
	config.Turn_stamina = 10
	config.Vision = defs.RisqVision{Space: 3, Edge_adjacent: 3, Adjacent: 3, Edge_opposite: 3, Secondary: 3}
	if len(ranges) > 0 {
		config.Attack_range = ranges[0]
	}
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
	if enemy.space == (harness.Coord{X: 2}) {
		home += "," + fmt.Sprintf(spaceJSON, 1, 0, flat, "")
	}
	return home + "," + fmt.Sprintf(spaceJSON, enemy.space.X, enemy.space.Y, flat, phalanx)
}

func connectorTowerHurtsEnemy(t *testing.T, attack_range defs.RisqRange, hops int, target_zone harness.Coord) bool {
	t.Helper()
	log := captureTicks(t)
	coordinates := []harness.Coord{{X: -3}, {}, {X: 3}, {Y: 3}}
	spaces := make([]string, 0, len(coordinates))
	connections := `[{"from":[-3,0],"to":[0,0],"direction":0,"to_direction":3},{"from":[0,0],"to":[3,0],"direction":2,"to_direction":3},{"from":[3,0],"to":[0,3],"direction":2,"to_direction":3}]`
	for i, coordinate := range coordinates {
		zones := ""
		if i == 0 {
			zones = fmt.Sprintf(`{"x":1,"y":0,"building":{"id":%d,"player":0}},`+zoneJSON, watchtower, 0, 0, fmt.Sprintf(unitJSON, 11, 0))
		}
		if i == hops {
			zones = fmt.Sprintf(zoneJSON, target_zone.X, target_zone.Y, fmt.Sprintf(unitJSON, heavy, 1))
		}
		spaces = append(spaces, fmt.Sprintf(spaceJSON, coordinate.X, coordinate.Y, flat, zones))
	}
	doc := fmt.Sprintf(`{"board_size":4,"players":2,"spaces":[%s],"connections":%s}`, strings.Join(spaces, ","), connections)
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"connector-range": doc})
	config := defs.BuildingConfigs[watchtower]
	config.Attack_range, config.Turn_stamina = attack_range, 10
	defs.BuildingConfigs[watchtower] = config
	g := harness.NewGameWithSettings(t, "custom:connector-range", 1, 2, map[string]any{"visibility": float64(defs.VisibilityMode_ALL_VISIBLE)})
	foe := g.Human(1)
	passive(g, foe, g.Self(foe).Units[0].InternalID)
	g.EndTurn()
	return len(parseTicks(t, log).hits) > 0
}

func TestZoneRangesFollowConnectorChains(t *testing.T) {
	cases := []struct {
		name       string
		range_type defs.RisqRange
		hops       int
		zone       harness.Coord
		hurt       bool
	}{
		{"wall follows connector", defs.RisqRange_WALL, 1, west, true},
		{"space follows connector", defs.RisqRange_SPACE, 1, west, true},
		{"twozone reaches second step", defs.RisqRange_TWOZONE, 1, harness.Coord{Y: -1}, true},
		{"twozone misses third step", defs.RisqRange_TWOZONE, 2, west, false},
		{"threezone crosses two connectors", defs.RisqRange_THREEZONE, 2, west, true},
		{"adjacent retains threezone", defs.RisqRange_ADJACENT, 2, west, true},
		{"fivezone crosses three connectors", defs.RisqRange_FIVEZONE, 3, west, true},
		{"fivezone misses sixth step", defs.RisqRange_FIVEZONE, 3, center, false},
		{"secondary retains fivezone", defs.RisqRange_SECONDARY, 3, west, true},
		{"secondary misses third-space sixth step", defs.RisqRange_SECONDARY, 3, center, false},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if hurt := connectorTowerHurtsEnemy(t, test.range_type, test.hops, test.zone); hurt != test.hurt {
				t.Errorf("hurt=%v, want %v", hurt, test.hurt)
			}
		})
	}
}

func TestZoneRangeBuildingReach(t *testing.T) {
	cases := []struct {
		name       string
		range_type defs.RisqRange
		tower      harness.Coord
		enemy      enemyPlacement
		hurt       bool
	}{
		{"wall center reaches edge", defs.RisqRange_WALL, center, sameWest, true},
		{"wall edge misses far edge", defs.RisqRange_WALL, east, sameWest, false},
		{"wall crosses border", defs.RisqRange_WALL, east, eastFacing, true},
		{"wall misses second step", defs.RisqRange_WALL, east, eastCenter, false},
		{"twozone reaches second step", defs.RisqRange_TWOZONE, east, eastCenter, true},
		{"twozone center crosses border", defs.RisqRange_TWOZONE, center, eastFacing, true},
		{"twozone misses third step", defs.RisqRange_TWOZONE, center, eastCenter, false},
		{"threezone reaches third step", defs.RisqRange_THREEZONE, center, eastCenter, true},
		{"fivezone reaches second border", defs.RisqRange_FIVEZONE, center, enemyPlacement{harness.Coord{X: 2}, west}, true},
		{"fivezone misses sixth step", defs.RisqRange_FIVEZONE, center, enemyPlacement{harness.Coord{X: 2}, center}, false},
		{"secondary reaches whole second space", defs.RisqRange_SECONDARY, center, enemyPlacement{harness.Coord{X: 2}, center}, true},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if hurt := towerHurtsEnemy(t, test.tower, test.enemy, test.range_type); hurt != test.hurt {
				t.Errorf("hurt=%v, want %v", hurt, test.hurt)
			}
		})
	}
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
