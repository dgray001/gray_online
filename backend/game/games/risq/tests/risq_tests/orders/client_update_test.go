package orders

import (
	"testing"

	"github.com/dgray001/gray_online/game"
	"github.com/gin-gonic/gin"
)

func applyClientUpdate(t *testing.T, client gin.H, update *game.UpdateMessage) gin.H {
	t.Helper()
	if snapshot, ok := update.Content["game"].(gin.H); ok {
		return snapshot
	}
	if update.Kind != "unit-behavior-set" {
		return client
	}
	for _, player := range client["players"].([]gin.H) {
		for _, unit := range player["units"].([]gin.H) {
			for _, id := range update.Content["internal_ids"].([]uint64) {
				if unit["internal_id"] != id {
					continue
				}
				for _, field := range []string{"stance", "interrupt_current", "attack_back", "target_priority"} {
					if value, exists := update.Content[field]; exists {
						unit[field] = value
					}
				}
			}
		}
	}
	return client
}
