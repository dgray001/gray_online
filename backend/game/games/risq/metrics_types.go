package risq

type StaminaMetrics struct {
	Granted int
	Wasted  int
}

// Opt-in ("production"): completed buildings of one type that produce units or techs, other than village centers
type ProductionMetrics struct {
	BuildingId uint32
	Count      int
	Stamina    StaminaMetrics
}

type PlayerTurnMetrics struct {
	Production           []ProductionMetrics `json:",omitempty"`
	PlayerId             int
	Stockpile            [4]float64
	Income               [4]float64
	Spent                [4]float64
	Villagers            int
	VillageCenters       int
	VillagerStamina      StaminaMetrics
	VillageCenterStamina StaminaMetrics
}

type TurnMetrics struct {
	Turn    uint16
	Players []PlayerTurnMetrics
}

type FightPlayerMetrics struct {
	Engaging                                                  []uint64
	Buildings                                                 []BuildingLocationMetrics
	NotAttacking                                              []uint64
	PlayerId                                                  int
	Military, Adjacent                                        map[uint32]int
	Units, Attacking, Moving, Exhausted, Garrisoned           []uint64
	AttackStamina, OverkillStamina, MoveStamina, OtherStamina float64
	IdleStamina                                               int
	Health                                                    float64
	Combatants                                                int
}

type BuildingLocationMetrics struct {
	InternalId uint64
	BuildingId uint32
	Space      [2]int
}

type FightTickMetrics struct {
	FightId    int
	Turn, Tick uint16
	Spaces     [][2]int
	Players    []FightPlayerMetrics
}

type FightMetrics struct {
	FightId int
	Ticks   []FightTickMetrics
}
