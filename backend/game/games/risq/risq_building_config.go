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
	display_name       string
	description        string
	max_health         int
	population_support uint16
	garrison_capacity  uint16
	produces           []Producible
	cost               RisqResourceCost
	build_stamina      int
	turn_stamina       int
	defense_blunt      int
	defense_piercing   int
	vision             RisqVision
	required_tech_id   uint32
	gather             BuildingGatherable
}

func (c BuildingConfig) isGatherable() bool {
	return c.gather.gather_capacity > 0
}

type BuildingGatherable struct {
	resource_category  RisqResourceCategory
	base_gather_speed  int
	starting_resources float64
	gather_capacity    int
	renew_cost         RisqResourceCost
	renew_stamina      int
}

type buildingGatherableJSON struct {
	ResourceCategory  string   `json:"resource_category"`
	BaseGatherSpeed   int      `json:"base_gather_speed"`
	StartingResources float64  `json:"starting_resources"`
	GatherCapacity    int      `json:"gather_capacity"`
	RenewCost         costJSON `json:"renew_cost"`
	RenewStamina      int      `json:"renew_stamina"`
}

func resolveGatherable(j *buildingGatherableJSON, source string) (BuildingGatherable, error) {
	if j == nil {
		return BuildingGatherable{}, nil
	}
	category, err := parseResourceCategory(j.ResourceCategory)
	if err != nil {
		return BuildingGatherable{}, fmt.Errorf("%s: %v", source, err)
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
		renew_stamina: j.RenewStamina,
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
	BuildingId        uint32                  `json:"building_id"`
	DisplayName       string                  `json:"display_name"`
	Description       string                  `json:"description"`
	MaxHealth         int                     `json:"max_health"`
	PopulationSupport uint16                  `json:"population_support"`
	GarrisonCapacity  uint16                  `json:"garrison_capacity"`
	Produces          []producibleJSON        `json:"produces"`
	Cost              costJSON                `json:"cost"`
	BuildStamina      int                     `json:"build_stamina"`
	TurnStamina       int                     `json:"turn_stamina"`
	DefenseBlunt      int                     `json:"defense_blunt"`
	DefensePiercing   int                     `json:"defense_piercing"`
	Vision            *risqVisionJSON         `json:"vision"`
	RequiredTechId    uint32                  `json:"required_tech_id"`
	Gatherable        *buildingGatherableJSON `json:"gatherable"`
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
			build_stamina:    e.BuildStamina,
			turn_stamina:     e.TurnStamina,
			defense_blunt:    e.DefenseBlunt,
			defense_piercing: e.DefensePiercing,
			vision:           resolveVision(e.Vision),
			required_tech_id: e.RequiredTechId,
			gather:           gather,
		}
	}
}
