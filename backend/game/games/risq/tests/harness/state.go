package harness

type Coord struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type Unit struct {
	InternalID     uint64  `json:"internal_id"`
	PlayerID       int     `json:"player_id"`
	UnitID         uint32  `json:"unit_id"`
	Space          Coord   `json:"space_coordinate"`
	Zone           Coord   `json:"zone_coordinate"`
	CurrentStamina int     `json:"current_stamina"`
	TurnStamina    int     `json:"turn_stamina"`
	GarrisonedIn   *uint64 `json:"garrisoned_in"`
	ActiveOrders   []struct {
		InternalID uint64 `json:"internal_id"`
		OrderType  uint8  `json:"order_type"`
	} `json:"active_orders"`
	CombatStats struct {
		Health       float64 `json:"health"`
		MaxHealth    int     `json:"max_health"`
		AttackBlunt  int     `json:"attack_blunt"`
		DefenseBlunt int     `json:"defense_blunt"`
	} `json:"combat_stats"`
	// The route planned on the last tick, for the owner only
	MovePath []struct {
		Space Coord `json:"space"`
		Zone  Coord `json:"zone"`
	} `json:"move_path"`
}

type Building struct {
	CurrentStamina    int     `json:"current_stamina"`
	InternalID        uint64  `json:"internal_id"`
	PlayerID          int     `json:"player_id"`
	BuildingID        uint32  `json:"building_id"`
	Space             Coord   `json:"space_coordinate"`
	Zone              Coord   `json:"zone_coordinate"`
	UnderConstruction bool    `json:"under_construction"`
	ResourcesLeft     float64 `json:"resources_left"`
	StaminaRemaining  int     `json:"stamina_remaining"`
	ProductionQueue   []struct {
		ItemID           uint32 `json:"item_id"`
		StaminaRemaining int    `json:"stamina_remaining"`
	} `json:"production_queue"`
	CombatStats struct {
		Health    float64 `json:"health"`
		MaxHealth int     `json:"max_health"`
	} `json:"combat_stats"`
}

type Resources struct {
	Food  float64 `json:"food"`
	Wood  float64 `json:"wood"`
	Stone float64 `json:"stone"`
	Gold  float64 `json:"gold"`
}

type PlayerState struct {
	Player struct {
		PlayerID int `json:"player_id"`
	} `json:"player"`
	Resources       *Resources      `json:"resources"`
	Units           []Unit          `json:"units"`
	Buildings       []Building      `json:"buildings"`
	OrdersSubmitted bool            `json:"orders_submitted"`
	Eliminated      bool            `json:"eliminated"`
	PopulationLimit int             `json:"population_limit"`
	ResearchedTechs map[uint32]bool `json:"researched_techs"`
	ActiveOrders    []struct {
		InternalID uint64 `json:"internal_id"`
		OrderType  uint8  `json:"order_type"`
	} `json:"active_orders"`
	PlannedFoundations []struct {
		CoordinateKey uint   `json:"coordinate_key"`
		BuildingID    uint32 `json:"building_id"`
	} `json:"planned_foundations"`
	AvailableMercenaries []struct {
		ID uint32 `json:"id"`
	} `json:"available_mercenaries"`
	// Why orders were refused last turn, for the player themself only
	TurnReport *struct {
		Orders struct {
			Failures []struct {
				Reason string `json:"reason"`
			} `json:"failures"`
		} `json:"orders"`
	} `json:"turn_report"`
}

// The reasons the last turn's orders were refused
func (p PlayerState) Refusals() []string {
	var reasons []string
	if p.TurnReport != nil {
		for _, f := range p.TurnReport.Orders.Failures {
			reasons = append(reasons, f.Reason)
		}
	}
	return reasons
}

type ZoneState struct {
	TerrainOverride uint32 `json:"terrain_override"`
	Coordinate      Coord  `json:"coordinate"`
	Ownership       *int   `json:"ownership"`
	Resource        *struct {
		ResourceID    uint32  `json:"resource_id"`
		ResourcesLeft float64 `json:"resources_left"`
	} `json:"resource"`
	Building  *Building `json:"building"`
	Units     []Unit    `json:"units"`
	UnitCount *int      `json:"unit_count"`
}

// Visibility is 0 unexplored, 1 fog, 2 poor, 3 good, 4 spy; hidden fields are absent, so their pointers stay nil
type SpaceState struct {
	Coordinate Coord         `json:"coordinate"`
	Visibility int           `json:"visibility"`
	TerrainID  *uint32       `json:"terrain_id"`
	Ownership  *int          `json:"ownership"`
	GoldIncome float64       `json:"gold_income"`
	UnitCount  *int          `json:"unit_count"`
	Zones      [][]ZoneState `json:"zones"`
	Units      []Unit        `json:"units"`
	Buildings  []Building    `json:"buildings"`
	Resources  []struct {
		ResourceID    uint32  `json:"resource_id"`
		ResourcesLeft float64 `json:"resources_left"`
	} `json:"resources"`
}

type Region struct {
	Name      string  `json:"name"`
	GoldBonus float64 `json:"gold_bonus"`
	Spaces    []uint  `json:"spaces"`
	Owner     int     `json:"owner"`
}

// A seam or link between spaces that are not normal coordinate neighbors; a nil direction is a center link
type SpaceLink struct {
	From      Coord `json:"from"`
	To        Coord `json:"to"`
	Direction int   `json:"direction"`
}

type State struct {
	TurnNumber   int             `json:"turn_number"`
	GivingOrders bool            `json:"giving_orders"`
	Players      []PlayerState   `json:"players"`
	Spaces       [][]*SpaceState `json:"spaces"`
	Regions      []Region        `json:"regions"`
	SpaceLinks   []SpaceLink     `json:"space_links"`
}

func (s State) Player(playerID int) *PlayerState {
	for i := range s.Players {
		if s.Players[i].Player.PlayerID == playerID {
			return &s.Players[i]
		}
	}
	return nil
}

// The space at a coordinate, nil when it was never on the board
func (s State) Space(x, y int) *SpaceState {
	for _, row := range s.Spaces {
		for _, space := range row {
			if space != nil && space.Coordinate.X == x && space.Coordinate.Y == y {
				return space
			}
		}
	}
	return nil
}

func (s State) SpaceCount() int {
	count := 0
	for _, row := range s.Spaces {
		for _, space := range row {
			if space != nil {
				count++
			}
		}
	}
	return count
}

// A unit anywhere in the visible state, nil when no player's payload lists it
func (s State) Unit(id uint64) *Unit {
	for _, p := range s.Players {
		for i := range p.Units {
			if p.Units[i].InternalID == id {
				return &p.Units[i]
			}
		}
	}
	return nil
}
