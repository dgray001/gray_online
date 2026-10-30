package aibridge

import (
	"encoding/json"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/gin-gonic/gin"
)

// Typed mirror of the per-player frontend game payload; the ai sees exactly what a human client would.
type snapGame struct {
	TurnNumber int            `json:"turn_number"`
	Players    []snapPlayer   `json:"players"`
	Spaces     [][]*snapSpace `json:"spaces"`
	Regions    []snapRegion   `json:"regions"`
	// adjacencies other than the normal border between coordinate neighbors
	SpaceLinks []snapSpaceLink `json:"space_links"`
}

type snapSpaceLink struct {
	From snapCoordinate `json:"from"`
	To   snapCoordinate `json:"to"`

	Direction   int `json:"direction"`
	ToDirection int `json:"to_direction"`
}

// Only the explored spaces of a region are listed
type snapRegion struct {
	Name          string  `json:"name"`
	GoldBonus     float64 `json:"gold_bonus"`
	Owner         int     `json:"owner"`
	Spaces        []uint  `json:"spaces"`
	HasUnexplored bool    `json:"has_unexplored_spaces"`
}

type snapCoordinate struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type snapCost struct {
	Food  float64 `json:"food"`
	Wood  float64 `json:"wood"`
	Stone float64 `json:"stone"`
	Gold  float64 `json:"gold"`
}

type snapOrder struct {
	InternalId uint64         `json:"internal_id"`
	OrderType  defs.OrderType `json:"order_type"`
	TargetId   int64          `json:"target_id"`
}

type snapPlayer struct {
	Player struct {
		PlayerId int    `json:"player_id"`
		Nickname string `json:"nickname"`
	} `json:"player"`
	Score                uint             `json:"score"`
	Eliminated           bool             `json:"eliminated"`
	PopulationLimit      int              `json:"population_limit"`
	MaxPopulationLimit   int              `json:"max_population_limit"`
	Resources            snapCost         `json:"resources"`
	PlannedFoundations   []snapFoundation `json:"planned_foundations"`
	Buildings            []snapBuilding   `json:"buildings"`
	Units                []snapUnit       `json:"units"`
	ActiveOrders         []snapOrder      `json:"active_orders"`
	ResearchedTechs      map[uint32]bool  `json:"researched_techs"`
	AvailableMercenaries []snapProducible `json:"available_mercenaries"`
}

type snapProducible struct {
	ID       uint32        `json:"id"`
	Kind     uint8         `json:"kind"`
	UnitType defs.UnitType `json:"unit_type"`
	Cost     snapCost      `json:"cost"`
	Stamina  int           `json:"stamina_cost"`
}

type snapFoundation struct {
	CoordinateKey uint   `json:"coordinate_key"`
	BuildingId    uint32 `json:"building_id"`
}

type snapUnit struct {
	AttackRange      defs.RisqRange  `json:"attack_range"`
	InternalId       uint64          `json:"internal_id"`
	PlayerId         int             `json:"player_id"`
	UnitId           uint32          `json:"unit_id"`
	UnitType         defs.UnitType   `json:"unit_type"`
	CurrentStamina   int             `json:"current_stamina"`
	MaxStamina       int             `json:"max_stamina"`
	GarrisonedIn     *uint64         `json:"garrisoned_in"`
	Zone             snapCoordinate  `json:"zone_coordinate"`
	Space            snapCoordinate  `json:"space_coordinate"`
	ActiveOrders     []snapOrder     `json:"active_orders"`
	Stance           defs.UnitStance `json:"stance"`
	InterruptCurrent bool            `json:"interrupt_current"`
	AttackBack       bool            `json:"attack_back"`
	TargetPriority   []int           `json:"target_priority"`
	CombatStats      struct {
		Health    float64 `json:"health"`
		MaxHealth int     `json:"max_health"`
	} `json:"combat_stats"`
}

type snapBuilding struct {
	InternalId        uint64         `json:"internal_id"`
	PlayerId          int            `json:"player_id"`
	BuildingId        uint32         `json:"building_id"`
	UnderConstruction bool           `json:"under_construction"`
	GarrisonCapacity  int            `json:"garrison_capacity"`
	GarrisonedUnits   []uint64       `json:"garrisoned_units"`
	AttackRange       defs.RisqRange `json:"attack_range"`
	MaxStamina        int            `json:"max_stamina"`
	ResourcesLeft     float64        `json:"resources_left"`
	Renewing          bool           `json:"renewing"`
	BaseGatherSpeed   int            `json:"base_gather_speed"`
	GatherCapacity    int            `json:"gather_capacity"`
	RenewCost         *snapCost      `json:"renew_cost"`
	Zone              snapCoordinate `json:"zone_coordinate"`
	Space             snapCoordinate `json:"space_coordinate"`
	ActiveOrders      []snapOrder    `json:"active_orders"`
	AutoAttack        bool           `json:"auto_attack"`
	InterruptCurrent  bool           `json:"interrupt_current"`
	TargetPriority    []int          `json:"target_priority"`
	CombatStats       struct {
		Health    float64 `json:"health"`
		MaxHealth int     `json:"max_health"`
	} `json:"combat_stats"`
}

type snapResource struct {
	ResourceId      uint32  `json:"resource_id"`
	InternalId      uint64  `json:"internal_id"`
	ResourcesLeft   float64 `json:"resources_left"`
	BaseGatherSpeed int     `json:"base_gather_speed"`
	GatherCapacity  int     `json:"gather_capacity"`
}

type snapZone struct {
	Coordinate    snapCoordinate `json:"coordinate"`
	CoordinateKey uint           `json:"coordinate_key"`
	Resource      *snapResource  `json:"resource"`
	Building      *snapBuilding  `json:"building"`
}

type snapTerrainStats struct {
	IntraCost          uint    `json:"intra_cost"`
	InterCost          uint    `json:"inter_cost"`
	BuildSpeedModifier float64 `json:"build_speed_modifier"`
	Impassable         bool    `json:"impassable"`
}

type snapSpace struct {
	TerrainType  uint8             `json:"terrain_type"`
	TerrainStats *snapTerrainStats `json:"terrain_stats"`

	UnitCount     *int           `json:"unit_count"`
	Coordinate    snapCoordinate `json:"coordinate"`
	CoordinateKey uint           `json:"coordinate_key"`
	Visibility    uint8          `json:"visibility"`
	Ownership     *int           `json:"ownership"`
	GoldIncome    float64        `json:"gold_income"`
	Zones         [][]snapZone   `json:"zones"`
	// buildings in the space: live ones when visible, the last-seen ones when fogged
	Buildings []snapBuilding `json:"buildings"`
}

func parseAiSnapshot(game gin.H) (*snapGame, error) {
	data, err := json.Marshal(game)
	if err != nil {
		return nil, err
	}
	var snapshot snapGame
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, err
	}
	return &snapshot, nil
}
