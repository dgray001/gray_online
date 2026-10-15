package defs

// Config is one complete set of stat tables; a game reads only from the Config it was created with
type Config struct {
	Units               map[uint32]UnitConfig
	Buildings           map[uint32]BuildingConfig
	Techs               map[uint32]TechConfig
	Resources           map[uint32]ResourceConfig
	Terrains            map[uint32]TerrainConfig
	RegionNames         []string
	LowestInterMoveCost uint
	lowestIntraMoveCost uint
	terrainMoveCosts    map[TerrainType]TerrainMoveCost
	terrainIdsByType    map[TerrainType][]uint32
	// a pointer so the default config follows reassignment of the exported BonusConfigs
	bonuses *[]TechConfig
}

func newConfig() *Config {
	return &Config{lowestIntraMoveCost: 1, LowestInterMoveCost: 1, bonuses: new([]TechConfig)}
}

// Bonuses applying to every player of the game
func (c *Config) Bonuses() []TechConfig {
	return *c.bonuses
}

// The tables LoadConfig installed, shared rather than copied so edits to the exported tables stay visible
func DefaultConfig() *Config {
	return &Config{
		Units:               UnitConfigs,
		Buildings:           BuildingConfigs,
		Techs:               TechConfigs,
		Resources:           ResourceConfigs,
		Terrains:            TerrainConfigs,
		bonuses:             &BonusConfigs,
		RegionNames:         regionNames,
		LowestInterMoveCost: LowestInterMoveCost,
		lowestIntraMoveCost: lowestIntraMoveCost,
		terrainMoveCosts:    terrainMoveCosts,
		terrainIdsByType:    terrainIdsByType,
	}
}

func (c *Config) installAsDefault() {
	UnitConfigs, BuildingConfigs, TechConfigs = c.Units, c.Buildings, c.Techs
	ResourceConfigs, TerrainConfigs, BonusConfigs = c.Resources, c.Terrains, c.Bonuses()
	regionNames, terrainMoveCosts, terrainIdsByType = c.RegionNames, c.terrainMoveCosts, c.terrainIdsByType
	lowestIntraMoveCost, LowestInterMoveCost = c.lowestIntraMoveCost, c.LowestInterMoveCost
}
