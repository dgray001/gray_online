package euchre

import "github.com/gin-gonic/gin"

type EuchreTeam struct {
	players [2]*EuchrePlayer
	team_id uint8
	score   uint8
	tricks  uint8
}

func (t *EuchreTeam) toFrontend() gin.H {
	team := gin.H{
		"team_id": t.team_id,
		"score":   t.score,
		"tricks":  t.tricks,
	}
	player_ids := []int{}
	for _, player := range t.players {
		if player != nil {
			player_ids = append(player_ids, player.player.Player_id)
		}
	}
	team["player_ids"] = player_ids
	return team
}
