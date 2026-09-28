package risq

import "github.com/dgray001/gray_online/game/games/risq/internal/defs"

func showOrdersTo(subject_player_id int, zone *RisqZone, viewer_player_id int) bool {
	if subject_player_id == viewer_player_id {
		return true
	}
	if zone == nil || zone.space == nil {
		return false
	}
	return zone.space.getVisibility(viewer_player_id) >= defs.VisibilitySpy
}
