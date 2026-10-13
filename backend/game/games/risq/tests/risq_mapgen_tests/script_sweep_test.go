package risq_mapgen_tests

import (
	"fmt"
	"os"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

// The board a script of only the shape step makes, so dimensions are checked without generating a whole map
func shapeOnlyBoard(t *testing.T, name string, size defs.MapSize) *fakeboard.Board {
	t.Helper()
	board, err := fakeboard.GenerateWithSize("script:"+name, 2, size, 1)
	if err == nil || err.Error() != startsMissing {
		t.Fatalf("size %d: error %v, want %q", size, err, startsMissing)
	}
	return board
}

// Sizes also generated with the full script, since terrain, resources and starts cost far more than the shape
func fullyGenerated(size defs.MapSize) bool {
	return size == defs.MapSize_MINUSCULE || size == defs.MapSize_MEDIUM || size == defs.MapSize_GIGANTIC
}

func shapeScriptNamed(kind string) string {
	return shapeScript(`{"kind":"` + kind + `"}`)
}

func TestHexagonMapSizeRadii(t *testing.T) {
	script, err := os.ReadFile(testConfig + "/maps/scripted/hexagon.json")
	if err != nil {
		t.Fatal(err)
	}
	useScripts(t, map[string]string{"hexagon": string(script), "hexagon_shape": shapeScriptNamed("hexagon")})
	for i, radius := range []uint16{4, 5, 6, 7, 7, 8, 8, 9, 10} {
		size := defs.MapSize(i + 1)
		spaces := 1 + 3*int(radius)*(int(radius)+1)
		if shape := shapeOnlyBoard(t, "hexagon_shape", size); shape.BoardSize() != radius || len(shape.Spaces()) != spaces {
			t.Errorf("size %d: radius %d, spaces %d; want %d, %d", size, shape.BoardSize(), len(shape.Spaces()), radius, spaces)
		}
		if !fullyGenerated(size) {
			continue
		}
		board, err := fakeboard.GenerateWithSize("script:hexagon", 2, size, 1)
		if err != nil {
			t.Fatalf("size %d: %v", size, err)
		}
		wantDistance := int(radius) - 1
		for player := 0; player < 2; player++ {
			_, _, home := board.Tally(player)
			var x, y int
			if _, err := fmt.Sscanf(home, "%d,%d", &x, &y); err != nil {
				t.Fatal(err)
			}
			if distance := max(x, -x, y, -y, x+y, -x-y); distance != wantDistance {
				t.Errorf("size %d: home %s distance %d, want %d", size, home, distance, wantDistance)
			}
		}
		checkBoardInvariants(t, board, 2)
	}
}

func TestRingMapSizeRadii(t *testing.T) {
	script, err := os.ReadFile(testConfig + "/maps/scripted/ring.json")
	if err != nil {
		t.Fatal(err)
	}
	useScripts(t, map[string]string{"ring": string(script), "ring_shape": shapeScriptNamed("ring")})
	radii := [][2]uint16{{4, 2}, {5, 2}, {5, 2}, {6, 3}, {7, 3}, {7, 3}, {8, 4}, {9, 4}, {10, 5}}
	for i, pair := range radii {
		outer, inner := int(pair[0]), int(pair[1])
		spaces := 3 * (outer*(outer+1) - inner*(inner+1))
		if shape := shapeOnlyBoard(t, "ring_shape", defs.MapSize(i+1)); shape.BoardSize() != pair[0] || len(shape.Spaces()) != spaces {
			t.Errorf("size %d: radius %d, spaces %d; want %d, %d", i+1, shape.BoardSize(), len(shape.Spaces()), outer, spaces)
		}
		if !fullyGenerated(defs.MapSize(i + 1)) {
			continue
		}
		board, err := fakeboard.GenerateWithSize("script:ring", 2, defs.MapSize(i+1), 1)
		if err != nil {
			t.Fatalf("size %d: %v", i+1, err)
		}
		checkBoardInvariants(t, board, 2)
		wantDistance := outer - 1
		for player := 0; player < 2; player++ {
			_, _, home := board.Tally(player)
			var x, y int
			if _, err := fmt.Sscanf(home, "%d,%d", &x, &y); err != nil {
				t.Fatal(err)
			}
			if distance := max(x, -x, y, -y, x+y, -x-y); distance != wantDistance {
				t.Errorf("size %d: home %s distance %d, want %d", i+1, home, distance, wantDistance)
			}
		}
	}
}

func TestRectangleMapSizeDimensions(t *testing.T) {
	script, err := os.ReadFile(testConfig + "/maps/scripted/rectangle.json")
	if err != nil {
		t.Fatal(err)
	}
	useScripts(t, map[string]string{"rectangle": string(script), "rectangle_shape": shapeScriptNamed("rectangle")})
	dimensions := [][2]int{{6, 9}, {8, 12}, {9, 14}, {10, 15}, {11, 16}, {12, 18}, {13, 19}, {14, 21}, {15, 23}}
	for i, want := range dimensions {
		if fullyGenerated(defs.MapSize(i + 1)) {
			board, err := fakeboard.GenerateWithSize("script:rectangle", 2, defs.MapSize(i+1), 1)
			if err != nil {
				t.Fatalf("size %d: %v", i+1, err)
			}
			checkBoardInvariants(t, board, 2)
		}
		board := shapeOnlyBoard(t, "rectangle_shape", defs.MapSize(i+1))
		rowCounts := map[int]int{}
		for _, space := range board.Inspect() {
			rowCounts[space.Coord.Y]++
		}
		if len(rowCounts) != want[0] {
			t.Errorf("size %d: rows %d, want %d", i+1, len(rowCounts), want[0])
		}
		for row, count := range rowCounts {
			if count != want[1] {
				t.Errorf("size %d row %d: columns %d, want %d", i+1, row, count, want[1])
			}
		}
	}
}

func TestTriangleMapSizeDimensions(t *testing.T) {
	script, err := os.ReadFile(testConfig + "/maps/scripted/triangle.json")
	if err != nil {
		t.Fatal(err)
	}
	useScripts(t, map[string]string{"triangle": string(script), "triangle_shape": shapeScriptNamed("triangle")})
	dimensions := [][2]int{{9, 55}, {12, 91}, {14, 120}, {16, 153}, {17, 171}, {19, 210}, {20, 231}, {23, 300}, {25, 351}}
	for i, want := range dimensions {
		if fullyGenerated(defs.MapSize(i + 1)) {
			board, err := fakeboard.GenerateWithSize("script:triangle", 2, defs.MapSize(i+1), 1)
			if err != nil {
				t.Fatalf("size %d: %v", i+1, err)
			}
			checkBoardInvariants(t, board, 2)
		}
		board := shapeOnlyBoard(t, "triangle_shape", defs.MapSize(i+1))
		rows := map[int]bool{}
		for _, space := range board.Inspect() {
			rows[space.Coord.Y] = true
		}
		if len(rows) != want[0]+1 || len(board.Spaces()) != want[1] {
			t.Errorf("size %d: %d rows and %d spaces, want %d and %d", i+1, len(rows), len(board.Spaces()), want[0]+1, want[1])
		}
	}
}

func TestScriptsGenerateForSweptPlayerCounts(t *testing.T) {
	t.Parallel()
	for _, name := range sweptScripts {
		for _, players := range sweptPlayerCounts(name) {
			t.Run(fmt.Sprintf("%s/p%d", name, players), func(t *testing.T) {
				board, err := fakeboard.Generate("script:"+name, players, 1)
				if err != nil {
					t.Fatal(err)
				}
				checkBoardInvariants(t, board, players)
			})
		}
	}
}
