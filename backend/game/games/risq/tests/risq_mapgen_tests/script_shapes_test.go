package risq_mapgen_tests

import "testing"

func TestShapesCarveExpectedSpaceCounts(t *testing.T) {
	cases := map[string]struct {
		params string
		want   int
	}{
		"hexagon0":      {`{"kind":"hexagon","size":0}`, 1},
		"hexagon3":      {`{"kind":"hexagon","size":3}`, 37},
		"hexagon_expr":  {`{"kind":"hexagon","size":"1 + 2 * 3"}`, 169},
		"ring":          {`{"kind":"ring","size":3,"inner_size":1}`, 30},
		"ring_thick":    {`{"kind":"ring","size":3,"thickness":2}`, 30},
		"rectangle_3x4": {`{"kind":"rectangle","rows":3,"cols":4}`, 12},
		"rectangle_1x1": {`{"kind":"rectangle","rows":1,"cols":1}`, 1},
		"triangle3":     {`{"kind":"triangle","size":3}`, 10},
	}
	scripts := map[string]string{}
	for name, c := range cases {
		scripts[name] = shapeScript(c.params)
	}
	useScripts(t, scripts)
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if got := len(generateStartless(t, name, 2).Inspect()); got != c.want {
				t.Errorf("%d spaces, want %d", got, c.want)
			}
		})
	}
}

func TestRecommendedHexagonSizeFollowsPlayerCount(t *testing.T) {
	useScripts(t, map[string]string{"rec": shapeScript(`{"kind":"hexagon","size":"recommended"}`)})
	for players, want := range map[int]int{2: 61, 3: 61, 4: 91, 5: 127, 6: 127, 12: 271} {
		if got := len(generateStartless(t, "rec", players).Inspect()); got != want {
			t.Errorf("%d players: %d spaces, want %d", players, got, want)
		}
	}
}
