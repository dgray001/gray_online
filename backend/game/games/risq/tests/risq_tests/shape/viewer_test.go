package shape

import "testing"

func TestViewerDoesNotInheritClientPrivateData(t *testing.T) {
	g := shapeGame(t, 4)
	queueWork(g)
	for human := range 2 {
		state := checkShape(t, snapshot(g, human, true), gameShape)
		equal(t, state["last_turn_replay"], nil)
		checkShape(t, state["game_base"], "game_id:n game_type:n game_started:b game_ended:b persistant_history:b players:a viewers:a player_actions:a viewer_updates:a")
		for _, value := range array(t, state["players"]) {
			p := checkShape(t, value, playerShape)
			for _, key := range []string{"units", "buildings", "active_orders"} {
				equal(t, len(array(t, p[key])), 0)
			}
		}
		for _, row := range array(t, state["spaces"]) {
			for _, value := range array(t, row) {
				if value != nil {
					s := checkShape(t, value, unexploredShape)
					equal(t, s["visibility"], float64(0))
				}
			}
		}
		equal(t, len(array(t, state["regions"])), 0)
	}
}
