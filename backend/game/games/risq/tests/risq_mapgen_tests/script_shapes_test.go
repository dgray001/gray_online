package risq_mapgen_tests

import "testing"

func TestRectangleCenteredOnHexagon(t *testing.T) {
	useScripts(t, map[string]string{"rectangle": shapeScript(`{"kind":"rectangle","size":{"rows":6,"cols":10}}`)})
	board := generateStartless(t, "rectangle", 2)
	minX, maxX, minY, maxY := 0, 0, 0, 0
	for i, space := range board.Inspect() {
		x, y := 2*space.Coord.X+space.Coord.Y, space.Coord.Y
		if i == 0 {
			minX, maxX, minY, maxY = x, x, y, y
		}
		minX, maxX = min(minX, x), max(maxX, x)
		minY, maxY = min(minY, y), max(maxY, y)
	}
	if max(minX+maxX, -minX-maxX) > 1 || max(minY+maxY, -minY-maxY) > 1 {
		t.Errorf("rectangle bounds are off-center: x [%d,%d], y [%d,%d]", minX, maxX, minY, maxY)
	}
	if board.BoardSize() != 6 || len(board.Spaces()) != 60 {
		t.Errorf("rectangle has radius %d and %d spaces, want 6 and 60", board.BoardSize(), len(board.Spaces()))
	}
}

func TestShapesCarveExpectedSpaceCounts(t *testing.T) {
	cases := map[string]struct {
		params string
		want   int
	}{
		"hexagon_default":   {`{"kind":"hexagon"}`, 61},
		"hexagon_empty":     {`{"kind":"hexagon","size":{}}`, 61},
		"hexagon_null":      {`{"kind":"hexagon","size":null}`, 61},
		"rectangle_partial": {`{"kind":"rectangle","size":{"rows":3}}`, 27},
		"hexagon0":          {`{"kind":"hexagon","size":{"radius":0}}`, 1},
		"hexagon3":          {`{"kind":"hexagon","size":{"radius":3}}`, 37},
		"hexagon_expr":      {`{"kind":"hexagon","size":{"radius":"1 + 2 * 3"}}`, 169},
		"ring":              {`{"kind":"ring","size":{"outer_radius":3,"inner_radius":1}}`, 30},
		"ring_partial":      {`{"kind":"ring","size":{"inner_radius":1}}`, 54},
		"rectangle_3x4":     {`{"kind":"rectangle","size":{"rows":3,"cols":4}}`, 12},
		"rectangle_1x1":     {`{"kind":"rectangle","size":{"rows":1,"cols":1}}`, 1},
		"triangle3":         {`{"kind":"triangle","size":{"edge_length":3}}`, 10},
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

func TestDefaultHexagonSizeFollowsResolvedMapSize(t *testing.T) {
	useScripts(t, map[string]string{"default": shapeScript(`{"kind":"hexagon"}`)})
	for players, want := range map[int]int{2: 61, 3: 91, 4: 127, 5: 169, 6: 169, 12: 331} {
		if got := len(generateStartless(t, "default", players).Inspect()); got != want {
			t.Errorf("%d players: %d spaces, want %d", players, got, want)
		}
	}
}
