package shape

import "testing"

func TestGameAndPlayerWireShapes(t *testing.T) {
	g := shapeGame(t, 3)
	for human := range 2 {
		state := checkShape(t, snapshot(g, human, false), gameShape)
		base := checkShape(t, state["game_base"], "game_id:n game_type:n game_started:b game_ended:b persistant_history:b players:a viewers:a")
		for _, value := range array(t, base["players"]) {
			checkShape(t, value, identityShape)
		}
		for _, value := range array(t, state["players"]) {
			p := object(t, value)
			checkShape(t, p["player"], identityShape)
			contract := playerShape
			if object(t, p["player"])["player_id"] == float64(g.PlayerID(human)) {
				contract += " " + privatePlayerShape + " default_unit_stance:n default_unit_attack_back:b default_unit_interrupt_current:b default_unit_target_priority:a"
				checkShape(t, p["resources"], costShape)
				mercenaries := array(t, p["available_mercenaries"])
				equal(t, len(mercenaries), 2)
				for _, value := range mercenaries {
					checkProducible(t, value)
				}
			}
			checkShape(t, p, contract)
			equal(t, p["researched_techs"], map[string]any{"4": true})
		}
		for _, key := range []string{"background_top_left", "background_top_right"} {
			equal(t, state[key], []any{float64(0), float64(0)})
		}
	}
}
