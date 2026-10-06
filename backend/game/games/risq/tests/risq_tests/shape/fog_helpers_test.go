package shape

import (
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func leaveTarget(g *harness.Game) {
	g.T.Helper()
	human, id := g.Human(0), unitID(g, 0, -1, 15)
	g.Submit(human, harness.OrderMove([]uint64{id}, -3, 0))
	for turn := 0; turn < 8; turn++ {
		g.EndTurn()
		if g.State(human).Unit(id).Space == (harness.Coord{X: -3}) {
			return
		}
	}
	g.T.Fatal("scout never left target vision")
}

const cachedBuildingShape = "internal_id:n player_id:n building_id:n display_name:s population_support:n combat_stats:o under_construction:b stamina_remaining:n construction_stamina_total:n turn_stamina:n current_stamina:n max_stamina:n produces:a active_orders:a production_queue:a zone_coordinate:o space_coordinate:o"
