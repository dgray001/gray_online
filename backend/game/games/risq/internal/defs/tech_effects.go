package defs

import (
	"bytes"
	"encoding/json"
	"fmt"
)

func (tech TechConfig) Effects() []TechConfig {
	return append([]TechConfig{tech}, tech.Additional_effects...)
}

func (tech TechConfig) UnlockedMercenaryIds() []uint32 {
	ids := make([]uint32, 0)
	for _, effect := range tech.Effects() {
		ids = append(ids, effect.Unlocks_mercenary_ids...)
	}
	return ids
}

func parseAdditionalTechEffects(entries []json.RawMessage, source string, tech_id uint32) []TechConfig {
	effects := make([]TechConfig, 0, len(entries))
	for i, entry := range entries {
		var effect techEffectJSON
		decoder := json.NewDecoder(bytes.NewReader(entry))
		decoder.DisallowUnknownFields()
		effect_source := fmt.Sprintf("%s %d additional_effects[%d]", source, tech_id, i)
		if err := decoder.Decode(&effect); err != nil {
			panic(fmt.Sprintf("%s: %v", effect_source, err))
		}
		effects = append(effects, parseTechConfigEntry(techConfigJSON{techEffectJSON: effect}, effect_source))
	}
	return effects
}

func (c *Config) ActiveTechEffects(tech_ids []uint32) []TechConfig {
	effects := make([]TechConfig, 0)
	for _, bonus := range c.Bonuses() {
		effects = append(effects, bonus.Effects()...)
	}
	for _, id := range tech_ids {
		if tech, ok := c.Techs[id]; ok {
			effects = append(effects, tech.Effects()...)
		}
	}
	return effects
}
