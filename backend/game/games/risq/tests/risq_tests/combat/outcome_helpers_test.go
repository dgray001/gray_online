package combat

import (
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
)

func checkOutcome(t *testing.T, snapshot gin.H, winners []int) {
	t.Helper()
	outcome, ok := snapshot["outcome"].(gin.H)
	if !ok || !reflect.DeepEqual(outcome["winner_player_ids"], winners) {
		t.Fatalf("outcome = %v, want winners %v", snapshot["outcome"], winners)
	}
	if snapshot["game_base"].(gin.H)["game_ended"] != true || snapshot["giving_orders"] != false {
		t.Fatalf("final snapshot is still running: %v", snapshot)
	}
	for _, player := range snapshot["players"].([]gin.H) {
		if len(player["active_orders"].([]gin.H)) != 0 {
			t.Fatal("final snapshot contains player orders")
		}
		for _, field := range []string{"units", "buildings"} {
			for _, entity := range player[field].([]gin.H) {
				if len(entity["active_orders"].([]gin.H)) != 0 {
					t.Fatal("final snapshot contains entity orders")
				}
			}
		}
	}
}
