package risq

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
)

//go:embed config/buildings.json
var buildingsConfigJSON []byte

type BuildingConfig struct {
	display_name                 string
	description                  string
	max_health                   int
	population_support           uint16
	garrison_capacity            uint16
	produces                     []Producible
	cost                         RisqResourceCost
	build_stamina                int
	turn_stamina                 int
	attack_type                  AttackType
	attack_blunt                 int
	attack_piercing              int
	attack_range                 RisqRange
	defense_blunt                int
	defense_piercing             int
	penetration_blunt            int
	penetration_piercing         int
	vision                       RisqVision
	required_tech_id             uint32
	gather                       BuildingGatherable
	max_garrison_attack_blunt    *int
	max_garrison_attack_piercing *int
	max_garrison_attack_magic    *int
	terrain_override             map[TerrainType]uint32
}

func (c BuildingConfig) isGatherable() bool {
	return c.gather.gather_capacity > 0
}

func (c BuildingConfig) canHaveGatherPoint() bool {
	if c.garrison_capacity > 0 {
		return true
	}
	for _, p := range c.produces {
		if p.kind == ProducibleKind_UNIT {
			return true
		}
	}
	return false
}

type BuildingGatherable struct {
	resource_category     RisqResourceCategory
	base_gather_speed     int
	starting_resources    float64
	gather_capacity       int
	renew_cost            RisqResourceCost
	renew_stamina         int
	terrain_override_dead map[TerrainType]uint32
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
		terrain_type, err := parseTerrainType(name)
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
		resource_category:  category,
		base_gather_speed:  j.BaseGatherSpeed,
		starting_resources: j.StartingResources,
		gather_capacity:    j.GatherCapacity,
		renew_cost: RisqResourceCost{
			food:  j.RenewCost.Food,
			wood:  j.RenewCost.Wood,
			stone: j.RenewCost.Stone,
			gold:  j.RenewCost.Gold,
		},
		renew_stamina:         j.RenewStamina,
		terrain_override_dead: terrain_override_dead,
	}, nil
}

func (c BuildingConfig) canProduce(unit_id uint32) bool {
	for _, p := range c.produces {
		if p.kind == ProducibleKind_UNIT && p.id == unit_id {
			return true
		}
	}
	return false
}

func (c BuildingConfig) canResearch(tech_id uint32) bool {
	for _, p := range c.produces {
		if p.kind == ProducibleKind_TECH && p.id == tech_id {
			return true
		}
	}
	return false
}

// Returns the resource cost and total build stamina required for a unit to construct this building type
func buildingProductionCost(building_id uint32) (RisqResourceCost, int) {
	config, ok := buildingConfigs[building_id]
	if !ok || config.build_stamina <= 0 {
		fmt.Fprintln(os.Stderr, "Unknown or unbuildable building id: ", building_id)
		return RisqResourceCost{}, 0
	}
	return config.cost, config.build_stamina
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

var buildingConfigs map[uint32]BuildingConfig

func init() {
	var entries []buildingConfigJSON
	if err := json.Unmarshal(buildingsConfigJSON, &entries); err != nil {
		panic(fmt.Sprintf("failed to parse config/buildings.json: %v", err))
	}
	buildingConfigs = make(map[uint32]BuildingConfig, len(entries))
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
		buildingConfigs[e.BuildingId] = BuildingConfig{
			display_name:       e.DisplayName,
			description:        e.Description,
			max_health:         e.MaxHealth,
			population_support: e.PopulationSupport,
			garrison_capacity:  e.GarrisonCapacity,
			produces:           produces,
			cost: RisqResourceCost{
				food:  e.Cost.Food,
				wood:  e.Cost.Wood,
				stone: e.Cost.Stone,
				gold:  e.Cost.Gold,
			},
			build_stamina:                e.BuildStamina,
			turn_stamina:                 e.TurnStamina,
			attack_type:                  attack_type,
			attack_blunt:                 e.AttackBlunt,
			attack_piercing:              e.AttackPiercing,
			attack_range:                 attack_range,
			defense_blunt:                e.DefenseBlunt,
			defense_piercing:             e.DefensePiercing,
			penetration_blunt:            e.PenetrationBlunt,
			penetration_piercing:         e.PenetrationPiercing,
			vision:                       resolveVision(e.Vision),
			required_tech_id:             e.RequiredTechId,
			gather:                       gather,
			max_garrison_attack_blunt:    e.MaxGarrisonAttackBlunt,
			max_garrison_attack_piercing: e.MaxGarrisonAttackPiercing,
			max_garrison_attack_magic:    e.MaxGarrisonAttackMagic,
			terrain_override:             terrain_override,
		}
	}
}
