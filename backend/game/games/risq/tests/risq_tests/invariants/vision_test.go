package invariants

import "github.com/dgray001/gray_online/game/games/risq/internal/defs"

func allSeeingConfig() {
	vision := defs.RisqVision{Space: 4, Edge_adjacent: 4, Adjacent: 4, Edge_opposite: 4, Secondary: 4}
	for id, config := range defs.UnitConfigs {
		config.Vision = vision
		defs.UnitConfigs[id] = config
	}
	for id, config := range defs.BuildingConfigs {
		config.Vision = vision
		defs.BuildingConfigs[id] = config
	}
}
