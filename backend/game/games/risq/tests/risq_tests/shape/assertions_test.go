package shape

import (
	"reflect"
	"strings"
	"testing"
)

func checkShape(t *testing.T, value any, contract string) map[string]any {
	t.Helper()
	result := object(t, value)
	fields := strings.Fields(contract)
	if len(result) != len(fields) {
		t.Errorf("want keys %s, got %v", contract, reflect.ValueOf(result).MapKeys())
	}
	for _, field := range fields {
		key, kind, _ := strings.Cut(field, ":")
		item, exists := result[key]
		if !exists || !jsonKind(item, kind) {
			t.Errorf("field %s: want JSON kind %s, got %T (%v), present %t", key, kind, item, item, exists)
		}
	}
	return result
}

func jsonKind(value any, kind string) bool {
	switch kind {
	case "n":
		_, ok := value.(float64)
		return ok
	case "s":
		_, ok := value.(string)
		return ok
	case "b":
		_, ok := value.(bool)
		return ok
	case "o":
		_, ok := value.(map[string]any)
		return ok
	case "o?":
		return value == nil || jsonKind(value, "o")
	case "a":
		_, ok := value.([]any)
		return ok
	}
	return false
}

func equal(t *testing.T, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
