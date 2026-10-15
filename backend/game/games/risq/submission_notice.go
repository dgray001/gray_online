package risq

import (
	"github.com/dgray001/gray_online/game"
	"github.com/gin-gonic/gin"
)

// Tells every player that someone submitted or unsubmitted instead of resending the whole game. These ride the
// settings channel, which is not ordered against the snapshots, so each carries the turn and clients drop stale ones.
// Only the owner gets their order list and a stale stored snapshot; nobody else reads another player's submission.
func (r *GameRisq) broadcastSubmission(kind string, player_id int) {
	submitter := r.players[player_id]
	for _, other := range r.players {
		content := gin.H{
			"player_id":        player_id,
			"orders_submitted": submitter.orders_submitted,
			"giving_orders":    r.giving_orders,
			"turn_number":      r.turn_number,
		}
		if other == submitter {
			content["active_orders"] = submitter.activeOrdersToFrontend()
			r.sendSetting(other.player, kind, content)
		} else {
			other.player.AddSettingUpdate(&game.UpdateMessage{Kind: kind, Content: content})
		}
	}
}
