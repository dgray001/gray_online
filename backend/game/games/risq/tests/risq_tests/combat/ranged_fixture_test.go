package combat

import (
	"fmt"
	"strings"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
	"github.com/gin-gonic/gin"
)

func rangedGame(t *testing.T, distance int, nearbyVillager bool) *harness.Game {
	t.Helper()
	var spaces []string
	for x := 0; x <= distance; x++ {
		zones := ""
		if x == 0 {
			units := `{"id":11,"player":0,"count":1}`
			if nearbyVillager {
				units += `,{"id":1,"player":1,"count":1}`
			}
			zones = `{"x":0,"y":0,"units":[` + units + `]}`
		} else if x == distance {
			zones = unitZoneJSON(harness.Coord{}, 13, 1)
		}
		spaces = append(spaces, fmt.Sprintf(`{"x":%d,"y":0,"terrain":1,"zones":[%s]}`, x, zones))
	}
	doc := fmt.Sprintf(`{"board_size":3,"players":2,"spaces":[%s]}`, strings.Join(spaces, ","))
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"ranged": doc})
	config := defs.UnitConfigs[11]
	config.Attack_range = defs.RisqRange_ADJACENT
	config.Vision.Adjacent, config.Vision.Edge_opposite, config.Vision.Secondary = defs.VisibilityGood, defs.VisibilityGood, defs.VisibilityGood
	defs.UnitConfigs[11] = config
	g := harness.NewGame(t, "custom:ranged", 1, 2)
	for _, unit := range g.Self(g.Human(1)).Units {
		if unit.UnitID == 13 {
			g.Action(g.Human(1), "set-unit-behavior", gin.H{"internal_ids": []uint64{unit.InternalID}, "stance": uint8(defs.UnitStance_PASSIVE), "attack_back": false})
		}
	}
	return g
}
