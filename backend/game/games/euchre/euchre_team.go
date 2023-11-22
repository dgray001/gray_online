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
	return team
}
