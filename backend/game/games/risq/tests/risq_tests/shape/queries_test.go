package shape

import "testing"

func player(t *testing.T, state map[string]any, id int) map[string]any {
	t.Helper()
	for _, value := range array(t, state["players"]) {
		item := object(t, value)
		if object(t, item["player"])["player_id"] == float64(id) {
			return item
		}
	}
	t.Fatalf("player %d missing", id)
	return nil
}

func space(t *testing.T, state map[string]any, x, y int) map[string]any {
	t.Helper()
	rows := array(t, state["spaces"])
	return object(t, array(t, rows[y+4])[x-max(-4, -(4+y))])
}

func entity(t *testing.T, owner map[string]any, collection string, id uint64) map[string]any {
	t.Helper()
	for _, value := range array(t, owner[collection]) {
		item := object(t, value)
		if item["internal_id"] == float64(id) {
			return item
		}
	}
	t.Fatalf("%s %d missing", collection, id)
	return nil
}
