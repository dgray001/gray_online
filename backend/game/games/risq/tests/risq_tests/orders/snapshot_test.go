package orders

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
)

// Serializes a payload with every unordered collection sorted, so equal states compare equal
func snapshotJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var decoded any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	data, err = json.Marshal(normalizeSnapshot("", decoded))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func normalizeSnapshot(key string, value any) any {
	switch data := value.(type) {
	case map[string]any:
		for field, item := range data {
			data[field] = normalizeSnapshot(field, item)
		}
	case []any:
		for i, item := range data {
			data[i] = normalizeSnapshot("", item)
		}
		if slices.Contains([]string{"players", "units", "buildings", "resources", "subjects", "garrisoned_units", "planned_foundations", "available_mercenaries"}, key) {
			slices.SortFunc(data, func(a, b any) int { return strings.Compare(fmt.Sprint(a), fmt.Sprint(b)) })
		}
	}
	return value
}
