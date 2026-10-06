package risq_mapgen_tests

import (
	"strings"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

func TestBadMapNamesFail(t *testing.T) {
	cases := map[string]string{
		"":                `must start with "custom:" or "script:"`,
		"ring":            `must start with "custom:" or "script:"`,
		"wat:ring":        `must start with "custom:" or "script:"`,
		"script:missing":  `map script "missing"`,
		"custom:missing":  `custom map "missing"`,
		"script:../ring":  "no such file",
		"custom:":         "no such file",
		"script:default ": "no such file",
	}
	for name, want := range cases {
		_, err := fakeboard.Generate(name, 2, 1)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: error %v, want it to contain %q", name, err, want)
		}
	}
}

func TestMalformedCustomMapFails(t *testing.T) {
	useCustoms(t, map[string]string{"badjson": `{nope`})
	if _, err := fakeboard.Generate("custom:badjson", 2, 1); err == nil || !strings.Contains(err.Error(), `custom map "badjson": invalid character`) {
		t.Errorf("error %v, want a JSON syntax error naming the custom map", err)
	}
}
