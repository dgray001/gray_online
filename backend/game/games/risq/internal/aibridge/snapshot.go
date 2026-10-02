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
	Score              uint             `json:"score"`
	PopulationLimit    int              `json:"population_limit"`
	MaxPopulationLimit int              `json:"max_population_limit"`
	Resources          snapCost         `json:"resources"`
	PlannedFoundations []snapFoundation `json:"planned_foundations"`
	Buildings          []snapBuilding   `json:"buildings"`
	Units              []snapUnit       `json:"units"`
	ActiveOrders       []snapOrder      `json:"active_orders"`
	ResearchedTechs    map[uint32]bool  `json:"researched_techs"`
}

type snapFoundation struct {
	CoordinateKey uint   `json:"coordinate_key"`
	BuildingId    uint32 `json:"building_id"`
}

type snapUnit struct {
	InternalId       uint64          `json:"internal_id"`
	PlayerId         int             `json:"player_id"`
	UnitId           uint32          `json:"unit_id"`
	UnitType         defs.UnitType   `json:"unit_type"`
	CurrentStamina   int             `json:"current_stamina"`
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
	ResourcesLeft     float64        `json:"resources_left"`
	Renewing          bool           `json:"renewing"`
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

type snapSpace struct {
	UnitCount     *int           `json:"unit_count"`
	Coordinate    snapCoordinate `json:"coordinate"`
	CoordinateKey uint           `json:"coordinate_key"`
	Visibility    uint8          `json:"visibility"`
	Ownership     *int           `json:"ownership"`
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
