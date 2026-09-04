package risq

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed config/techs.json
var techsConfigJSON []byte

type TechConfig struct {
	display_name       string
	description        string
	cost               RisqResourceCost
	research_stamina   int
	affects_unit_id    uint32
	bonus_max_health   int
	bonus_turn_stamina int
}

type techConfigJSON struct {
	TechId           uint32   `json:"tech_id"`
	DisplayName      string   `json:"display_name"`
	Description      string   `json:"description"`
	Cost             costJSON `json:"cost"`
	ResearchStamina  int      `json:"research_stamina"`
	AffectsUnitId    uint32   `json:"affects_unit_id"`
	BonusMaxHealth   int      `json:"bonus_max_health"`
	BonusTurnStamina int      `json:"bonus_turn_stamina"`
}

var techConfigs map[uint32]TechConfig

func init() {
	var entries []techConfigJSON
	if err := json.Unmarshal(techsConfigJSON, &entries); err != nil {
		panic(fmt.Sprintf("failed to parse config/techs.json: %v", err))
	}
	techConfigs = make(map[uint32]TechConfig, len(entries))
	for _, e := range entries {
		techConfigs[e.TechId] = TechConfig{
			display_name: e.DisplayName,
			description:  e.Description,
			cost: RisqResourceCost{
				food:  e.Cost.Food,
				wood:  e.Cost.Wood,
				stone: e.Cost.Stone,
				gold:  e.Cost.Gold,
			},
			research_stamina:   e.ResearchStamina,
			affects_unit_id:    e.AffectsUnitId,
			bonus_max_health:   e.BonusMaxHealth,
			bonus_turn_stamina: e.BonusTurnStamina,
		}
	}
}

func applyTechBonus(u *RisqUnit, tech TechConfig) {
	u.turn_stamina += tech.bonus_turn_stamina
	u.cs.setMaxHealth(u.cs.max_health + tech.bonus_max_health)
}

func (r *GameRisq) completeResearch(player *RisqPlayer, tech_id uint32) {
	player.researched_techs[tech_id] = true
	tech, ok := techConfigs[tech_id]
	if !ok {
		return
	}
	for _, unit := range player.units {
		if unit.unit_id != tech.affects_unit_id {
			continue
		}
		applyTechBonus(unit, tech)
	}
}
