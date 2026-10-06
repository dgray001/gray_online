package aibridge

import (
	"sort"

	"github.com/dgray001/gray_online/game/games/risq/ai"
)

func (v *aiView) PlayerIDs() []int {
	ids := make([]int, 0, len(v.game.Players))
	for _, p := range v.game.Players {
		ids = append(ids, p.Player.PlayerId)
	}
	sort.Ints(ids)
	return ids
}

func (v *aiView) PlayerID() int {
	return v.playerId()
}

func (v *aiView) PlayerScore(player_id int) int {
	for _, p := range v.game.Players {
		if p.Player.PlayerId == player_id {
			return int(p.Score)
		}
	}
	return 0
}

func (v *aiView) AllZones() []ai.ZoneInfo {
	zones := make([]ai.ZoneInfo, 0, len(v.zones))
	for _, entry := range v.zones {
		info := ai.ZoneInfo{Location: entry.ref, HasResource: entry.zone.Resource != nil, BuildingPlayer: -1}
		if b := entry.zone.Building; b != nil {
			info.BuildingPlayer = b.PlayerId
		}
		zones = append(zones, info)
	}
	return zones
}
