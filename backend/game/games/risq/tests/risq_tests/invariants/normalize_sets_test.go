package invariants

import "testing"

func TestNormalizationIgnoresSetOrderAndLobbyIdentity(t *testing.T) {
	a := object{"player": object{"player_id": 0, "client_id": 1, "nickname": "first"}, "game_base": object{"time": 1}, "units": []any{object{"internal_id": 2}, object{"internal_id": 1}}, "regions": []any{object{"spaces": []any{2.0, 1.0}}}}
	b := object{"player": object{"player_id": 0, "client_id": 2, "nickname": "second"}, "game_base": object{"time": 2}, "units": []any{object{"internal_id": 1}, object{"internal_id": 2}}, "regions": []any{object{"spaces": []any{1.0, 2.0}}}}
	if canonical(normalize("", a)) != canonical(normalize("", b)) {
		t.Fatal("equivalent states normalized differently")
	}
	if len(a["units"].([]any)) != 2 || a["player"].(object)["player_id"] != 0 {
		t.Fatal("normalization discarded entities or slot identity")
	}
}
