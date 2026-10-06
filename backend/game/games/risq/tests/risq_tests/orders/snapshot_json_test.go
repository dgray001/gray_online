package orders

import (
	"encoding/json"
	"testing"
)

func snapshotJSON(t *testing.T, value any) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var decoded any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	encoded, err = json.Marshal(normalizeSnapshot("", decoded))
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}
