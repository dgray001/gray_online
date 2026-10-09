package risq

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
)

type AutoRenewFromFrontend struct {
	Building_id uint32 `json:"building_id"`
	Count       *int   `json:"count"`
}

func getAutoRenewFromPlayerAction(action gin.H) (AutoRenewFromFrontend, error) {
	var request AutoRenewFromFrontend
	data, err := json.Marshal(action)
	if err == nil {
		err = json.Unmarshal(data, &request)
	}
	return request, err
}

func (r *GameRisq) executeSetAutoRenew(player_id int, action gin.H) {
	p := r.players[player_id]
	request, err := getAutoRenewFromPlayerAction(action)
	if err != nil {
		p.player.AddFailedUpdateShorthand("set-auto-renew-failed", err.Error())
		return
	}
	if request.Count == nil {
		p.player.AddFailedUpdateShorthand("set-auto-renew-failed", "auto-renew count is required")
		return
	}
	if err := p.setAutoRenewCount(request.Building_id, *request.Count); err != nil {
		p.player.AddFailedUpdateShorthand("set-auto-renew-failed", err.Error())
		return
	}
	content := gin.H{"building_id": request.Building_id, "count": *request.Count}
	p.player.AddUpdate(r.playerSnapshotUpdate(p.player, "auto-renew-set", content))
}
