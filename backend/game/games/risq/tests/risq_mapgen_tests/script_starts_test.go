package risq_mapgen_tests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

func startBuilding(distance, edge int) string {
	return fmt.Sprintf(`{"building_id":2,"target":{"space_distance":%d,"zone":"edge","edge_index":%d}}`, distance, edge)
}

// Start areas of radius 2 two apart overlap, so mirrored building zones are often already taken by a neighbor
func overlappingStarts() string {
	buildings := []string{`{"building_id":1,"target":{"zone":"center"}}`}
	for _, spot := range [][2]int{{1, 1}, {1, 2}, {1, 3}, {2, 1}, {2, 2}, {2, 3}, {1, 4}, {1, 5}} {
		buildings = append(buildings, startBuilding(spot[0], spot[1]))
	}
	return `[` + step("shape", `{"kind":"hexagon","size":4}`) + `,` + grass + `,` + step("player_starts",
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
