package risq

import "github.com/dgray001/gray_online/game/games/risq/internal/defs"

func (r *GameRisq) initializeVisibility() {
	level := r.visibility_mode.MinimumVision()
	if level == defs.VisibilityUnexplored {
		return
	}
	for _, space := range r.allSpaces() {
		for _, player := range r.players {
			player_id := player.player.Player_id
			space.visibility[player_id] = max(space.getVisibility(player_id), level)
			space.refreshCache(player_id)
		}
	}
}
