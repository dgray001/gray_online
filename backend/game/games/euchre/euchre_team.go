package euchre

import "github.com/gin-gonic/gin"

type EuchreTeam struct {
	players [2]*EuchrePlayer
	score   uint8
	tricks  uint8
}

func (p *EuchreTeam) toFrontend() gin.H {
	team := gin.H{
		"score":  p.score,
		"tricks": p.tricks,
	}
	return team
}
