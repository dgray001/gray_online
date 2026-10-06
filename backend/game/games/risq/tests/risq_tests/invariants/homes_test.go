package invariants

import (
	"fmt"
	"strings"
)

var homes = [][2]int{{-1, 0}, {1, 0}, {0, 1}}

func homeSpaces(players, workers int) string {
	var spaces []string
	for slot, home := range homes[:players] {
		spaces = append(spaces, fmt.Sprintf(`{"x":%d,"y":%d,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":1,"player":%d},"units":[{"id":1,"player":%d,"count":%d}]},{"x":1,"y":0,"resource":11}]}`, home[0], home[1], slot, slot, workers))
	}
	return strings.Join(spaces, ",")
}

const centralSpace = `,{"x":0,"y":0,"terrain":1}`
const testRegions = `,"regions":[{"name":"West","gold_bonus":7,"spaces":[[-1,0],[0,0]]},{"name":"East","gold_bonus":3,"spaces":[[1,0]]}]`

func pairedCenters(workers int) string {
	spaces := strings.Replace(homeSpaces(2, workers), `"resource":11`, `"building":{"id":1,"player":0}`, 1)
	return strings.Replace(spaces, `"resource":11`, `"building":{"id":1,"player":1}`, 1) + centralSpace
}
