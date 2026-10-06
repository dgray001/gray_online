package risq

import (
	"github.com/dgray001/gray_online/game"
	"github.com/gin-gonic/gin"
)

func (r *GameRisq) playerSnapshotUpdate(player *game.Player, kind string, content gin.H) *game.UpdateMessage {
	content["game"] = r.toFrontendFor(player.Player_id, player.GetClientId(), false)
	return &game.UpdateMessage{Kind: kind, Content: content}
}
