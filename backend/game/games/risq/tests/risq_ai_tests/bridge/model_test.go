package bridge

import (
	"encoding/json"
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"math/rand"
	"testing"
)

func parsedModel(t *testing.T, actions string) ai.Model {
	t.Helper()
	var raw map[string]any
	if err := json.Unmarshal([]byte(`{"rules":[{"when":{"always":{}},"then":[`+actions+`]}]}`), &raw); err != nil {
		t.Fatal(err)
	}
	model := ai.ParseModel(raw, rand.New(rand.NewSource(1)))
	if _, ok := model.(*ai.RulesModel); !ok {
		t.Fatal("invalid fixture model")
	}
	return model
}
