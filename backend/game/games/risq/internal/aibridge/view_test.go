package aibridge

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestKnownEnemyBuildingsIncludesFog(t *testing.T) {
	building := func(id uint64, player int, x, y int) gin.H {
		return gin.H{"internal_id": id, "player_id": player, "building_id": 1,
			"space_coordinate": gin.H{"x": x, "y": y}, "zone_coordinate": gin.H{"x": 0, "y": 0}}
	}
	payload := gin.H{
		"players": []gin.H{
			{"player": gin.H{"player_id": 0}, "buildings": []gin.H{building(1, 0, 0, 0)}},
			// the enemy's live list only holds what we can see right now
			{"player": gin.H{"player_id": 1}, "buildings": []gin.H{building(20, 1, 2, 0)}},
		},
		"spaces": [][]gin.H{{
			{"coordinate": gin.H{"x": 0, "y": 0}, "visibility": 3, "buildings": []gin.H{building(1, 0, 0, 0)}},
			{"coordinate": gin.H{"x": 2, "y": 0}, "visibility": 3, "buildings": []gin.H{building(20, 1, 2, 0)}},
			// fogged space: its last-seen enemy Village Center
			{"coordinate": gin.H{"x": 4, "y": 0}, "visibility": 1, "buildings": []gin.H{building(30, 1, 4, 0)}},
		}},
	}
	snapshot, err := parseAiSnapshot(payload)
	if err != nil {
		t.Fatal(err)
	}
	view, ok := newAiView(snapshot, 0)
	if !ok {
		t.Fatal("no view")
	}
	if got := len(view.VisibleEnemyBuildings()); got != 1 {
		t.Errorf("visible enemy buildings = %d, want 1", got)
	}
	known := view.KnownEnemyBuildings()
	if len(known) != 2 {
		t.Fatalf("known enemy buildings = %d, want 2 (visible + fogged, own excluded, no duplicates)", len(known))
	}
	ids := map[uint64]bool{known[0].InternalID: true, known[1].InternalID: true}
	if !ids[20] || !ids[30] {
		t.Errorf("known = %+v, want ids 20 and 30", known)
	}
}
