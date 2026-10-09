package risq_mapgen_tests

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

type startHome struct {
	player, x, y int
}

func startHomes(t *testing.T, board *fakeboard.Board, players int) []startHome {
	t.Helper()
	homes := make([]startHome, players)
	for player := range homes {
		_, _, home := board.Tally(player)
		homes[player].player = player
		if _, err := fmt.Sscanf(home, "%d,%d", &homes[player].x, &homes[player].y); err != nil {
			t.Fatal(err)
		}
	}
	return homes
}

func startColumn(home startHome) int {
	return home.x + int(math.Floor(float64(home.y)/2))
}

func triangleStartIndices(t *testing.T, board *fakeboard.Board, players, inset int) (int, []int) {
	t.Helper()
	spaces := board.Inspect()
	minX, minY, maxSum := spaces[0].Coord.X, spaces[0].Coord.Y, spaces[0].Coord.X+spaces[0].Coord.Y
	for _, space := range spaces {
		minX, minY = min(minX, space.Coord.X), min(minY, space.Coord.Y)
		maxSum = max(maxSum, space.Coord.X+space.Coord.Y)
	}
	edge := maxSum - minX - minY - 3*inset
	indices := make([]int, 0, players)
	for _, home := range startHomes(t, board, players) {
		x, y := home.x-minX-inset, home.y-minY-inset
		if x < 0 || y < 0 || x+y > edge {
			t.Fatalf("home %v violates triangle inset %d", home, inset)
		}
		switch {
		case y == 0:
			indices = append(indices, x)
		case x+y == edge:
			indices = append(indices, edge+y)
		case x == 0:
			indices = append(indices, 3*edge-y)
		default:
			t.Fatalf("home %v is not on the inset perimeter", home)
		}
	}
	sort.Ints(indices)
	return edge, indices
}

func TestTriangleFinalStartingSpacePositions(t *testing.T) {
	script, err := os.ReadFile("../../config/maps/scripted/triangle.json")
	if err != nil {
		t.Fatal(err)
	}
	var steps []map[string]any
	if err := json.Unmarshal(script, &steps); err != nil {
		t.Fatal(err)
	}
	for i, step := range steps {
		if step["step"] == "player_starts" {
			layout, err := json.Marshal(steps[:i+1])
			if err != nil {
				t.Fatal(err)
			}
			useScripts(t, map[string]string{"triangle": string(script), "start_layout": string(layout)})
			break
		}
	}
	for size := defs.MapSize_MINUSCULE; size <= defs.MapSize_GIGANTIC; size++ {
		for players := 1; players <= 12; players++ {
			for seed := int64(1); seed <= 16; seed++ {
				mapName := "script:start_layout"
				if size == defs.DefaultMapSize(players) {
					mapName = "script:triangle"
				}
				board, err := fakeboard.GenerateWithSize(mapName, players, size, seed)
				if err != nil {
					t.Fatalf("size %d, %d players seed %d: %v", size, players, seed, err)
				}
				inset := 1
				if size >= defs.MapSize_HUGE {
					inset = 2
				}
				edge, indices := triangleStartIndices(t, board, players, inset)
				if players == 1 && indices[0]%edge != 0 {
					t.Fatalf("single player is not at a corner: %v", indices)
				}
				if players%3 == 0 {
					for corner := 0; corner < 3; corner++ {
						index := sort.SearchInts(indices, corner*edge)
						if index == len(indices) || indices[index] != corner*edge {
							t.Fatalf("%d players: corner %d is missing from %v", players, corner, indices)
						}
					}
				}
				minGap, maxGap := 3*edge, 0
				for i, position := range indices {
					gap := indices[(i+1)%players] + 3*edge*((i+1)/players) - position
					minGap, maxGap = min(minGap, gap), max(maxGap, gap)
				}
				if minGap == 0 || maxGap-minGap > 1 {
					t.Fatalf("size %d, %d players seed %d: uneven perimeter gaps %d/%d at %v", size, players, seed, minGap, maxGap, indices)
				}
			}
		}
	}
}

func rowStartDistances(homes []startHome) map[int][]uint {
	sort.Slice(homes, func(i, j int) bool {
		if homes[i].y != homes[j].y {
			return homes[i].y < homes[j].y
		}
		return homes[i].x < homes[j].x
	})
	distances := map[int][]uint{}
	for i := 1; i < len(homes); i++ {
		a, b := homes[i-1], homes[i]
		if a.y == b.y {
			distance := game_utils.AxialDistance(game_utils.Coordinate2D{X: a.x, Y: a.y}, game_utils.Coordinate2D{X: b.x, Y: b.y})
			distances[a.y] = append(distances[a.y], distance)
		}
	}
	return distances
}

func TestRectangleFinalStartingSpaceDistances(t *testing.T) {
	script, err := os.ReadFile("../../config/maps/scripted/rectangle.json")
	if err != nil {
		t.Fatal(err)
	}
	var steps []map[string]any
	if err := json.Unmarshal(script, &steps); err != nil {
		t.Fatal(err)
	}
	for i, step := range steps {
		if step["step"] == "player_starts" {
			startScript, err := json.Marshal(steps[:i+1])
			if err != nil {
				t.Fatal(err)
			}
			useScripts(t, map[string]string{"rectangle": string(script), "start_layout": string(startScript)})
			break
		}
	}
	for size := defs.MapSize_MINUSCULE; size <= defs.MapSize_GIGANTIC; size++ {
		inset := 1
		if size >= defs.MapSize_LARGE {
			inset = 2
		}
		for players := 1; players <= 12; players++ {
			expectedShort := 0
			if size >= defs.MapSize_HUGE {
				expectedShort = min(2, players)
			}
			for seed := int64(1); seed <= 16; seed++ {
				mapName := "script:start_layout"
				if size == defs.DefaultMapSize(players) {
					mapName = "script:rectangle"
				}
				board, err := fakeboard.GenerateWithSize(mapName, players, size, seed)
				if err != nil {
					t.Fatalf("size %d, %d players seed %d: %v", size, players, seed, err)
				}
				homes := startHomes(t, board, players)
				minRow, maxRow, minCol, maxCol := 0, 0, 0, 0
				for i, space := range board.Inspect() {
					col := startColumn(startHome{x: space.Coord.X, y: space.Coord.Y})
					if i == 0 {
						minRow, maxRow = space.Coord.Y, space.Coord.Y
						minCol, maxCol = col, col
					}
					minRow, maxRow = min(minRow, space.Coord.Y), max(maxRow, space.Coord.Y)
					minCol, maxCol = min(minCol, col), max(maxCol, col)
				}
				rows := map[int]bool{}
				longHomes, shortHomes := []startHome{}, []startHome{}
				cornerStarts := 0
				for _, home := range homes {
					col := startColumn(home)
					if col < minCol+inset || col > maxCol-inset {
						t.Fatalf("size %d: home column %d violates inset %d", size, col, inset)
					}
					if col == minCol+inset || col == maxCol-inset {
						cornerStarts++
					}
					if home.y != minRow+inset && home.y != maxRow-inset {
						if home.y < minRow+inset || home.y > maxRow-inset || (col != minCol+inset && col != maxCol-inset) {
							t.Fatalf("size %d: invalid short-edge start %v", size, home)
						}
						shortHomes = append(shortHomes, home)
						continue
					}
					longHomes = append(longHomes, home)
					rows[home.y] = true
				}
				if len(rows) != min(players-expectedShort, 2) {
					t.Fatalf("size %d, %d players seed %d: starts occupy %d rows", size, players, seed, len(rows))
				}
				if cornerStarts != min(players-expectedShort, 2)+expectedShort {
					t.Fatalf("size %d, %d players seed %d: corner starts moved: %v", size, players, seed, homes)
				}
				if len(shortHomes) != expectedShort {
					t.Fatalf("size %d: short-edge starts %v, want %d", size, shortHomes, expectedShort)
				}
				if len(shortHomes) == 2 && startColumn(shortHomes[0]) == startColumn(shortHomes[1]) {
					t.Fatal("both short-edge starts are on the same edge")
				}
				nearCount, farCount := 0, 0
				for _, home := range longHomes {
					if home.y == minRow+inset {
						nearCount++
					} else {
						farCount++
					}
				}
				for _, home := range shortHomes {
					nearDistance, farDistance := home.y-(minRow+inset), maxRow-inset-home.y
					if (nearCount < farCount && nearDistance >= farDistance) || (farCount < nearCount && farDistance >= nearDistance) {
						t.Fatalf("short-edge start %v is not closer to the less-populated row (%d/%d players)", home, nearCount, farCount)
					}
					if nearCount == farCount && math.Abs(float64(nearDistance-farDistance)) > 1 {
						t.Fatalf("balanced rows: short-edge start %v is not centered", home)
					}
				}
				distances := rowStartDistances(longHomes)
				for row, gaps := range distances {
					minGap, maxGap := gaps[0], gaps[0]
					for _, gap := range gaps {
						minGap, maxGap = min(minGap, gap), max(maxGap, gap)
					}
					if minGap == 0 || maxGap-minGap > 1 {
						t.Fatalf("size %d, %d players seed %d row %d: uneven rounded distances %v", size, players, seed, row, gaps)
					}
				}
				if size == defs.MapSize_LARGE && players == 7 && seed == 1 {
					t.Logf("seven-player starting-space distances by row: %v", distances)
				}
				if size == defs.MapSize_LARGE && players == 7 {
					for row, gaps := range distances {
						if len(gaps) != 3 {
							continue
						}
						columns := []int{}
						for _, home := range homes {
							if home.y == row {
								columns = append(columns, startColumn(home)-minCol)
							}
						}
						sort.Ints(columns)
						if fmt.Sprint(columns) != "[2 6 11 15]" {
							t.Fatalf("seven-player corner/intermediate columns: %v", columns)
						}
						if seed == 1 {
							t.Logf("four-player row columns: %v", columns)
						}
					}
				}
			}
		}
	}
}

func startOrderKey(homes []startHome) string {
	sort.Slice(homes, func(i, j int) bool {
		return math.Atan2(float64(homes[i].y), float64(homes[i].x)) < math.Atan2(float64(homes[j].y), float64(homes[j].x))
	})
	zero := 0
	for i, home := range homes {
		if home.player == 0 {
			zero = i
		}
	}
	order := make([]int, len(homes))
	for i := range order {
		order[i] = homes[(zero+i)%len(homes)].player
	}
	return fmt.Sprint(order)
}

func TestStartAssignmentsAreShuffled(t *testing.T) {
	useStartLayouts(t)
	for _, shape := range []string{"hexagon", "ring", "rectangle", "triangle"} {
		orders := map[string]bool{}
		for seed := int64(1); seed <= 16; seed++ {
			board, err := fakeboard.Generate("script:"+shape, 4, seed)
			if err != nil {
				t.Fatal(err)
			}
			orders[startOrderKey(startHomes(t, board, 4))] = true
		}
		if len(orders) < 3 {
			t.Errorf("%s player assignments only rotate or reverse: %v", shape, orders)
		}
	}
}

func TestRectangleStartingRowAndSideAreRandom(t *testing.T) {
	useStartLayouts(t)
	corners := map[[2]int]bool{}
	for seed := int64(1); seed <= 16; seed++ {
		board, err := fakeboard.Generate("script:rectangle", 1, seed)
		if err != nil {
			t.Fatal(err)
		}
		home := startHomes(t, board, 1)[0]
		corners[[2]int{startColumn(home), home.y}] = true
	}
	if len(corners) != 4 {
		t.Errorf("starting row and side do not vary across all corners: %v", corners)
	}
}

func TestRectangleStartsZigzagForEveryPlayerCount(t *testing.T) {
	useStartLayouts(t)
	for players := 1; players <= 12; players++ {
		for seed := int64(1); seed <= 4; seed++ {
			board, err := fakeboard.Generate("script:rectangle", players, seed)
			if err != nil {
				t.Fatal(err)
			}
			homes := startHomes(t, board, players)
			sort.Slice(homes, func(i, j int) bool { return startColumn(homes[i]) < startColumn(homes[j]) })
			for i, home := range homes {
				if math.Abs(float64(home.y)) != 4 || (i%2 == 0) != (home.y == homes[0].y) {
					t.Errorf("%d players seed %d: starts do not alternate rows: %v", players, seed, homes)
				}
			}
			for row, gaps := range rowStartDistances(homes) {
				minGap, maxGap := gaps[0], gaps[0]
				for _, gap := range gaps {
					minGap, maxGap = min(minGap, gap), max(maxGap, gap)
				}
				if minGap == 0 || maxGap-minGap > 1 {
					t.Errorf("%d players seed %d row %d: uneven rounded distances %v", players, seed, row, gaps)
				}
			}
		}
	}
}

func useStartLayouts(t *testing.T) {
	starts := step("player_starts", `{"pattern":"%s","area_size":0,"starting_distance":4,"inset":1,"terrain_id":1,"resources":[],"buildings":[{"building_id":1,"target":{"zone":"center"}}]}`)
	scripts := map[string]string{}
	for name, params := range map[string]string{"hexagon": `{"kind":"hexagon","size":{"radius":5}}`, "ring": `{"kind":"ring","size":{"outer_radius":5,"inner_radius":2}}`, "rectangle": `{"kind":"rectangle","size":{"rows":12,"cols":23}}`, "triangle": `{"kind":"triangle","size":{"edge_length":9}}`} {
		pattern := "ring"
		if name == "rectangle" {
			pattern = "rows"
		}
		if name == "triangle" {
			pattern = "triangle"
		}
		scripts[name] = `[` + step("shape", params) + `,` + fmt.Sprintf(starts, pattern) + `]`
	}
	useScripts(t, scripts)
}

func TestRectangleRowInsetComesFromScript(t *testing.T) {
	script := `[` + step("shape", `{"kind":"rectangle","size":{"rows":12,"cols":23}}`) + `,` +
		step("player_starts", `{"pattern":"rows","area_size":0,"starting_distance":5,"row_inset":"map_size + 2","terrain_id":1,"resources":[],"buildings":[{"building_id":1,"target":{"zone":"center"}}]}`) + `]`
	useScripts(t, map[string]string{"inset": script})
	board, err := fakeboard.Generate("script:inset", 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, home := range startHomes(t, board, 2) {
		if home.y != -3 && home.y != 2 {
			t.Errorf("script inset 3 was not applied: home row %d", home.y)
		}
	}
}

func startBuilding(distance, edge int) string {
	return fmt.Sprintf(`{"building_id":2,"target":{"space_distance":%d,"zone":"edge","edge_index":%d}}`, distance, edge)
}

// Start areas of radius 2 two apart overlap, so mirrored building zones are often already taken by a neighbor
func overlappingStarts() string {
	buildings := []string{`{"building_id":1,"target":{"zone":"center"}}`}
	for _, spot := range [][2]int{{1, 1}, {1, 2}, {1, 3}, {2, 1}, {2, 2}, {2, 3}, {1, 4}, {1, 5}} {
		buildings = append(buildings, startBuilding(spot[0], spot[1]))
	}
	return `[` + step("shape", `{"kind":"hexagon","size":{"radius":4}}`) + `,` + grass + `,` + step("player_starts",
		`{"pattern":"ring","area_size":2,"starting_distance":2,"terrain_id":1,"resources":[],`+
			`"units":[{"unit_id":1,"count":1,"target":{"zone":"center"}}],"buildings":[`+strings.Join(buildings, ",")+`]}`) + `]`
}

func TestStartBuildingsFallBackWhenTheMirroredZoneIsTaken(t *testing.T) {
	useScripts(t, map[string]string{"overlap": overlappingStarts()})
	for players := 3; players <= 6; players++ {
		for seed := int64(1); seed <= 3; seed++ {
			board, err := fakeboard.Generate("script:overlap", players, seed)
			if err != nil {
				t.Fatalf("%d players seed %d: %v", players, seed, err)
			}
			for player := 0; player < players; player++ {
				if _, buildings, _ := board.Tally(player); buildings[1] != 1 || buildings[2] != 8 {
					t.Errorf("%d players seed %d: player %d has buildings %v, want 1 of id 1 and 8 of id 2", players, seed, player, buildings)
				}
			}
		}
	}
}
