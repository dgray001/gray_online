package invariants

import "github.com/dgray001/gray_online/game/games/risq/tests/harness"

func snapshots(g *harness.Game, players int) []object {
	g.T.Helper()
	views := make([]object, players)
	for slot := range views {
		views[slot] = jsonValue(g.T, g.Risq.ToFrontend(uint64(g.Human(slot)+1), false)).(object)
	}
	return views
}

func self(view object, slot int) object {
	for _, entry := range view["players"].([]any) {
		player := entry.(object)
		if player["player"].(object)["player_id"] == float64(slot) {
			return player
		}
	}
	panic("player missing from own snapshot")
}
