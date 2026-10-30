package defs

import (
	"fmt"
	"slices"

	"github.com/gin-gonic/gin"
)

type GatherBonus struct {
	Affects_resource_types  []RisqResourceCategory
	Affects_resource_ids    []uint32
	Bonus_gather_speed      int
	Bonus_gather_percentage float64
}

type gatherBonusJSON struct {
	AffectsResourceTypes  []string `json:"affects_resource_types"`
	AffectsResourceIds    []uint32 `json:"affects_resource_ids"`
	BonusGatherSpeed      int      `json:"bonus_gather_speed"`
	BonusGatherPercentage float64  `json:"bonus_gather_percentage"`
}

func (g GatherBonus) IsSet() bool {
	return g.Bonus_gather_speed != 0 || g.Bonus_gather_percentage != 0
}

func (g GatherBonus) AppliesTo(category RisqResourceCategory, resource_id uint32) bool {
	if len(g.Affects_resource_types) == 0 && len(g.Affects_resource_ids) == 0 {
		return true
	}
	return slices.Contains(g.Affects_resource_types, category) || (resource_id != 0 && slices.Contains(g.Affects_resource_ids, resource_id))
}

var resourceCategoryNames = map[RisqResourceCategory]string{
	RisqResourceCategory_FOOD:  "food",
	RisqResourceCategory_WOOD:  "wood",
	RisqResourceCategory_STONE: "stone",
	RisqResourceCategory_GOLD:  "gold",
}

func (g GatherBonus) ToFrontend(c *Config) gin.H {
	type_names := make([]string, len(g.Affects_resource_types))
	for i, category := range g.Affects_resource_types {
		type_names[i] = resourceCategoryNames[category]
	}
	resource_names := make([]string, len(g.Affects_resource_ids))
	for i, id := range g.Affects_resource_ids {
		resource_names[i] = c.Resources[id].Display_name
	}
	return gin.H{
		"resource_types": type_names,
		"resource_names": resource_names,
		"speed":          g.Bonus_gather_speed,
		"percentage":     g.Bonus_gather_percentage,
	}
}

func parseGatherBonus(e gatherBonusJSON) (GatherBonus, error) {
	if e.BonusGatherPercentage <= -100 {
		return GatherBonus{}, fmt.Errorf("bonus_gather_percentage must be above -100")
	}
	types := make([]RisqResourceCategory, len(e.AffectsResourceTypes))
	for i, s := range e.AffectsResourceTypes {
		category, err := parseResourceCategory(s)
		if err != nil {
			return GatherBonus{}, err
		}
		types[i] = category
	}
	return GatherBonus{
		Affects_resource_types:  types,
		Affects_resource_ids:    e.AffectsResourceIds,
		Bonus_gather_speed:      e.BonusGatherSpeed,
		Bonus_gather_percentage: e.BonusGatherPercentage,
	}, nil
}
