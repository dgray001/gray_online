package risq

import (
	"strings"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/gin-gonic/gin"
)

func (r *GameRisq) executeSetDefaultUnitBehavior(player_id int, behavior UnitBehaviorFromFrontend, kind string) {
	player := r.players[player_id]
	if behavior.Stance != nil {
		player.default_unit_stance = defs.UnitStance(*behavior.Stance)
	}
	if behavior.Attack_back != nil {
		player.default_unit_attack_back = *behavior.Attack_back
	}
	if behavior.Interrupt_current != nil {
		player.default_unit_interrupt_current = *behavior.Interrupt_current
	}
	if behavior.Target_priority != nil {
		player.default_unit_target_priority = make([]defs.TargetCategory, 0, len(*behavior.Target_priority))
		for _, raw := range *behavior.Target_priority {
			category := defs.TargetCategory(raw)
			if category > defs.TargetCategory_NONE && category < defs.TargetCategory_END {
				player.default_unit_target_priority = append(player.default_unit_target_priority, category)
			}
		}
	}
	player.player.AddUpdate(r.playerSnapshotUpdate(player.player, strings.TrimPrefix(kind, "set-")+"-set", gin.H{
		"stance":            player.default_unit_stance,
		"attack_back":       player.default_unit_attack_back,
		"interrupt_current": player.default_unit_interrupt_current,
		"target_priority":   targetCategoriesToInts(player.default_unit_target_priority),
	}))
}
