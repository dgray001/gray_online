package risq_mapgen_tests

import (
	"fmt"
	"os"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

func TestDefaultMapSizeRadii(t *testing.T) {
	script, err := os.ReadFile("../../config/maps/scripted/default.json")
	if err != nil {
		t.Fatal(err)
	}
	useScripts(t, map[string]string{"default": string(script)})
	for i, radius := range []uint16{4, 5, 6, 7, 7, 8, 8, 9, 10} {
		size := defs.MapSize(i + 1)
		board, err := fakeboard.GenerateWithSize("script:default", 2, size, 1)
		if err != nil {
			t.Fatalf("size %d: %v", size, err)
		}
		spaces := 1 + 3*int(radius)*(int(radius)+1)
		if board.BoardSize() != radius || len(board.Spaces()) != spaces {
			t.Errorf("size %d: radius %d, spaces %d; want %d, %d", size, board.BoardSize(), len(board.Spaces()), radius, spaces)
		}
		wantDistance := int(radius) - 1
		if radius >= 8 {
			wantDistance--
		}
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
	script, err := os.ReadFile("../../config/maps/scripted/ring.json")
	if err != nil {
		t.Fatal(err)
	}
	useScripts(t, map[string]string{"ring": string(script)})
	radii := [][2]uint16{{4, 2}, {5, 2}, {5, 2}, {6, 3}, {7, 3}, {7, 3}, {8, 4}, {9, 4}, {10, 5}}
	for i, pair := range radii {
		board, err := fakeboard.GenerateWithSize("script:ring", 2, defs.MapSize(i+1), 1)
		if err != nil {
			t.Fatalf("size %d: %v", i+1, err)
		}
		checkBoardInvariants(t, board, 2)
		outer, inner := int(pair[0]), int(pair[1])
		spaces := 3 * (outer*(outer+1) - inner*(inner+1))
		if board.BoardSize() != pair[0] || len(board.Spaces()) != spaces {
			t.Errorf("size %d: radius %d, spaces %d; want %d, %d", i+1, board.BoardSize(), len(board.Spaces()), outer, spaces)
		}
		wantDistance := outer - 1
		if outer-inner >= 5 {
			wantDistance--
		}
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
	script, err := os.ReadFile("../../config/maps/scripted/rectangle.json")
	if err != nil {
		t.Fatal(err)
	}
	useScripts(t, map[string]string{"rectangle": string(script)})
	dimensions := [][2]int{{6, 9}, {8, 12}, {9, 14}, {10, 15}, {11, 16}, {12, 18}, {13, 19}, {14, 21}, {15, 23}}
	for i, want := range dimensions {
		board, err := fakeboard.GenerateWithSize("script:rectangle", 2, defs.MapSize(i+1), 1)
		if err != nil {
			t.Fatalf("size %d: %v", i+1, err)
		}
		checkBoardInvariants(t, board, 2)
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

func TestScriptsGenerateForSweptPlayerCounts(t *testing.T) {
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
