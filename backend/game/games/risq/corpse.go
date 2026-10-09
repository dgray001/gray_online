package risq

import (
	"maps"
	"slices"

	"github.com/gin-gonic/gin"
)

type RisqCorpse struct {
	unit_id   uint32
	player_id int
	turns     uint8
}

func (z *RisqZone) advanceCorpses() {
	for id, corpse := range z.corpses {
		if corpse.turns == 2 {
			delete(z.corpses, id)
		} else {
			corpse.turns++
			z.corpses[id] = corpse
		}
	}
}

func (z *RisqZone) corpsesToFrontend() []gin.H {
	corpses := make([]gin.H, 0, len(z.corpses))
	for _, id := range slices.Sorted(maps.Keys(z.corpses)) {
		corpse := z.corpses[id]
		corpses = append(corpses, gin.H{
			"internal_id": id,
			"unit_id":     corpse.unit_id,
			"player_id":   corpse.player_id,
			"turns":       corpse.turns,
		})
	}
	return corpses
}
