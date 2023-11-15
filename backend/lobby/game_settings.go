package lobby

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

type GameSettings struct {
	MaxPlayers           uint8                  `json:"max_players"`
	MaxViewers           uint8                  `json:"max_viewers"`
	GameType             uint8                  `json:"game_type"`
	GameSpecificSettings map[string]interface{} `json:"game_specific_settings"`
}

func (s *GameSettings) Launchable() (bool, string) {
	if s.MaxPlayers < 1 || s.MaxPlayers > 8 {
		return false, "Invalid max players"
	}
	if s.MaxViewers < 0 || s.MaxViewers > 16 {
		return false, "Invalid max viewers"
	}
	if s.GameType != 1 {
		return false, "Invalid game type: " + strconv.Itoa(int(s.GameType))
	}
	return true, ""
}

func (s *GameSettings) ToFrontend() gin.H {
	return gin.H{
		"game_type":              strconv.Itoa(int(s.GameType)),
		"max_players":            strconv.Itoa(int(s.MaxPlayers)),
		"max_viewers":            strconv.Itoa(int(s.MaxViewers)),
		"game_specific_settings": s.GameSpecificSettings,
	}
}
