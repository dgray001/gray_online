package defs

import (
	"encoding/json"
	"fmt"
	"os"
)

type BuildingConfig struct {
	Display_name                 string
	Description                  string
	Max_health                   int
	Population_support           uint16
	Garrison_capacity            uint16
	Produces                     []Producible
	Cost                         RisqResourceCost
	Build_stamina                int
	Turn_stamina                 int
	Attack_type                  AttackType
	Attack_blunt                 int
	Attack_piercing              int
	Attack_range                 RisqRange
	Defense_blunt                int
	Defense_piercing             int
	Penetration_blunt            int
	Penetration_piercing         int
	Vision                       RisqVision
	Required_tech_id             uint32
	Gather                       BuildingGatherable
	Max_garrison_attack_blunt    *int
	Max_garrison_attack_piercing *int
	Max_garrison_attack_magic    *int
	Terrain_override             map[TerrainType]uint32
}

func (c BuildingConfig) IsGatherable() bool {
	return c.Gather.Gather_capacity > 0
}

func (c BuildingConfig) CanHaveGatherPoint() bool {
	if c.Garrison_capacity > 0 {
		return true
	}
	for _, p := range c.Produces {
		if p.Kind == ProducibleKind_UNIT {
			return true
		}
	}
	return false
}

type BuildingGatherable struct {
	Resource_category     RisqResourceCategory
	Base_gather_speed     int
	Starting_resources    float64
	Gather_capacity       int
	Renew_cost            RisqResourceCost
	Renew_stamina         int
	Terrain_override_dead map[TerrainType]uint32
}

type buildingGatherableJSON struct {
	ResourceCategory    string            `json:"resource_category"`
	BaseGatherSpeed     int               `json:"base_gather_speed"`
	StartingResources   float64           `json:"starting_resources"`
	GatherCapacity      int               `json:"gather_capacity"`
	RenewCost           costJSON          `json:"renew_cost"`
	RenewStamina        int               `json:"renew_stamina"`
	TerrainOverrideDead map[string]uint32 `json:"terrain_override_dead"`
}

// Converts a JSON terrain_type-name-keyed override map to one keyed by the parsed TerrainType
func resolveTerrainOverride(j map[string]uint32, source string) (map[TerrainType]uint32, error) {
	if len(j) == 0 {
		return nil, nil
	}
	override := make(map[TerrainType]uint32, len(j))
	for name, terrain_id := range j {
		terrain_type, err := ParseTerrainType(name)
		if err != nil {
			return nil, fmt.Errorf("%s: %v", source, err)
		}
		override[terrain_type] = terrain_id
	}
	return override, nil
}

func resolveGatherable(j *buildingGatherableJSON, source string) (BuildingGatherable, error) {
	if j == nil {
		return BuildingGatherable{}, nil
	}
	category, err := parseResourceCategory(j.ResourceCategory)
	if err != nil {
		return BuildingGatherable{}, fmt.Errorf("%s: %v", source, err)
	}
	terrain_override_dead, err := resolveTerrainOverride(j.TerrainOverrideDead, source)
	if err != nil {
		return BuildingGatherable{}, err
	}
	return BuildingGatherable{
		Resource_category:     category,
		Base_gather_speed:     j.BaseGatherSpeed,
		Starting_resources:    j.StartingResources,
		Gather_capacity:       j.GatherCapacity,
		Renew_cost:            j.RenewCost.toCost(),
		Renew_stamina:         j.RenewStamina,
		Terrain_override_dead: terrain_override_dead,
	}, nil
}

func (c BuildingConfig) CanProduce(unit_id uint32) bool {
	for _, p := range c.Produces {
		if p.Kind == ProducibleKind_UNIT && p.Id == unit_id {
			return true
		}
	}
	return false
}

func (c BuildingConfig) CanResearch(tech_id uint32) bool {
	for _, p := range c.Produces {
		if p.Kind == ProducibleKind_TECH && p.Id == tech_id {
			return true
		}
	}
	return false
}

// Returns the resource cost and total build stamina required for a unit to construct this building type
func BuildingProductionCost(building_id uint32) (RisqResourceCost, int) {
	config, ok := BuildingConfigs[building_id]
	if !ok || config.Build_stamina <= 0 {
		fmt.Fprintln(os.Stderr, "Unknown or unbuildable building id: ", building_id)
		return RisqResourceCost{}, 0
	}
	return config.Cost, config.Build_stamina
}

type buildingConfigJSON struct {
	BuildingId                uint32                  `json:"building_id"`
	DisplayName               string                  `json:"display_name"`
	Description               string                  `json:"description"`
	MaxHealth                 int                     `json:"max_health"`
	PopulationSupport         uint16                  `json:"population_support"`
	GarrisonCapacity          uint16                  `json:"garrison_capacity"`
	Produces                  []producibleJSON        `json:"produces"`
	Cost                      costJSON                `json:"cost"`
	BuildStamina              int                     `json:"build_stamina"`
	TurnStamina               int                     `json:"turn_stamina"`
	AttackType                string                  `json:"attack_type"`
	AttackBlunt               int                     `json:"attack_blunt"`
	AttackPiercing            int                     `json:"attack_piercing"`
	Range                     string                  `json:"range"`
	DefenseBlunt              int                     `json:"defense_blunt"`
	DefensePiercing           int                     `json:"defense_piercing"`
	PenetrationBlunt          int                     `json:"penetration_blunt"`
	PenetrationPiercing       int                     `json:"penetration_piercing"`
	Vision                    *risqVisionJSON         `json:"vision"`
	RequiredTechId            uint32                  `json:"required_tech_id"`
	Gatherable                *buildingGatherableJSON `json:"gatherable"`
	MaxGarrisonAttackBlunt    *int                    `json:"max_garrison_attack_blunt"`
	MaxGarrisonAttackPiercing *int                    `json:"max_garrison_attack_piercing"`
	MaxGarrisonAttackMagic    *int                    `json:"max_garrison_attack_magic"`
	TerrainOverride           map[string]uint32       `json:"terrain_override"`
}

var BuildingConfigs map[uint32]BuildingConfig

func loadBuildingConfig(data []byte) {
	var entries []buildingConfigJSON
	if err := json.Unmarshal(data, &entries); err != nil {
		panic(fmt.Sprintf("failed to parse config/buildings.json: %v", err))
	}
	BuildingConfigs = make(map[uint32]BuildingConfig, len(entries))
	for _, e := range entries {
		produces, err := parseProducibles(e.Produces, nil)
		if err != nil {
			panic(fmt.Sprintf("config/buildings.json building_id %d: %v", e.BuildingId, err))
		}
		gather, err := resolveGatherable(e.Gatherable, fmt.Sprintf("config/buildings.json building_id %d", e.BuildingId))
		if err != nil {
			panic(err)
		}
		attack_type, err := parseAttackType(e.AttackType)
		if err != nil {
			panic(fmt.Sprintf("config/buildings.json building_id %d: %v", e.BuildingId, err))
		}
		attack_range, err := parseRange(e.Range)
		if err != nil {
			panic(fmt.Sprintf("config/buildings.json building_id %d: %v", e.BuildingId, err))
		}
		terrain_override, err := resolveTerrainOverride(e.TerrainOverride, fmt.Sprintf("config/buildings.json building_id %d", e.BuildingId))
		if err != nil {
			panic(err)
		}
		BuildingConfigs[e.BuildingId] = BuildingConfig{
			Display_name:                 e.DisplayName,
			Description:                  e.Description,
			Max_health:                   e.MaxHealth,
			Population_support:           e.PopulationSupport,
			Garrison_capacity:            e.GarrisonCapacity,
			Produces:                     produces,
			Cost:                         e.Cost.toCost(),
			Build_stamina:                e.BuildStamina,
			Turn_stamina:                 e.TurnStamina,
			Attack_type:                  attack_type,
			Attack_blunt:                 e.AttackBlunt,
			Attack_piercing:              e.AttackPiercing,
			Attack_range:                 attack_range,
			Defense_blunt:                e.DefenseBlunt,
			Defense_piercing:             e.DefensePiercing,
			Penetration_blunt:            e.PenetrationBlunt,
			Penetration_piercing:         e.PenetrationPiercing,
			Vision:                       resolveVision(e.Vision),
			Required_tech_id:             e.RequiredTechId,
			Gather:                       gather,
			Max_garrison_attack_blunt:    e.MaxGarrisonAttackBlunt,
			Max_garrison_attack_piercing: e.MaxGarrisonAttackPiercing,
			Max_garrison_attack_magic:    e.MaxGarrisonAttackMagic,
			Terrain_override:             terrain_override,
		}
	}
}
