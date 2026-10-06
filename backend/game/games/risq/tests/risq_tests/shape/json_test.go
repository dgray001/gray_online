package shape

import (
	"encoding/json"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func snapshot(g *harness.Game, human int, viewer bool) map[string]any {
	g.T.Helper()
	data, err := json.Marshal(g.Risq.ToFrontend(uint64(human+1), viewer))
	if err != nil {
		g.T.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		g.T.Fatal(err)
	}
	return result
}

func object(t *testing.T, value any) map[string]any {
	t.Helper()
	result, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("want JSON object, got %T (%v)", value, value)
	}
	return result
}

func array(t *testing.T, value any) []any {
	t.Helper()
	result, ok := value.([]any)
	if !ok {
		t.Fatalf("want JSON array, got %T (%v)", value, value)
	}
	return result
}
