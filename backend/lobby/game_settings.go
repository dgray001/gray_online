package lobby

import (
	"strconv"

	"github.com/dgray001/gray_online/game"
	"github.com/gin-gonic/gin"
)

// set by main.go at startup
var DEV = false

type GameSettings struct {
	MaxPlayers           uint8                  `json:"max_players"`
	MaxViewers           uint8                  `json:"max_viewers"`
	GameType             game.GameType          `json:"game_type"`
	GameSpecificSettings map[string]interface{} `json:"game_specific_settings"`
}

func (s *GameSettings) Launchable() (bool, string) {
	max_game_type := game.GameType_RISQ
	if DEV {
		max_game_type = game.GameType_TEST_GAME
	}
	if s.GameType < 1 || s.GameType > max_game_type {
		return false, "Invalid game type: " + strconv.Itoa(int(s.GameType))
	}
	max_players := uint8(8)
	if s.GameType == game.GameType_RISQ {
		max_players = 12
	}
	if s.MaxPlayers < 1 || s.MaxPlayers > max_players {
		return false, "Invalid max players"
	}
	if s.MaxViewers > 16 {
		return false, "Invalid max viewers"
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
