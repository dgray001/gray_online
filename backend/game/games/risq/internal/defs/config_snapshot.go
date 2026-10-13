package defs

import "maps"

// A parsed copy of every config table, so tests can reset what they mutate without re-reading JSON
type ConfigSnapshot struct {
	buildings        map[uint32]BuildingConfig
	units            map[uint32]UnitConfig
	techs            map[uint32]TechConfig
	resources        map[uint32]ResourceConfig
	terrains         map[uint32]TerrainConfig
	terrainMoveCosts map[TerrainType]TerrainMoveCost
	terrainIdsByType map[TerrainType][]uint32
	bonuses          []TechConfig
	regionNames      []string
	lowestIntra      uint
	lowestInter      uint
}

func SnapshotConfig() *ConfigSnapshot {
	return &ConfigSnapshot{
		buildings:        maps.Clone(BuildingConfigs),
		units:            maps.Clone(UnitConfigs),
		techs:            maps.Clone(TechConfigs),
		resources:        maps.Clone(ResourceConfigs),
		terrains:         maps.Clone(TerrainConfigs),
		terrainMoveCosts: maps.Clone(terrainMoveCosts),
		terrainIdsByType: maps.Clone(terrainIdsByType),
		bonuses:          append([]TechConfig(nil), BonusConfigs...),
		regionNames:      append([]string(nil), regionNames...),
		lowestIntra:      lowestIntraMoveCost,
		lowestInter:      LowestInterMoveCost,
	}
}

// Replaces every table with a fresh copy of the snapshot and points file reads at dir
func (s *ConfigSnapshot) Restore(dir string) {
	configDir = dir
	BuildingConfigs = maps.Clone(s.buildings)
	UnitConfigs = maps.Clone(s.units)
	TechConfigs = maps.Clone(s.techs)
	ResourceConfigs = maps.Clone(s.resources)
	TerrainConfigs = maps.Clone(s.terrains)
	terrainMoveCosts = maps.Clone(s.terrainMoveCosts)
	terrainIdsByType = maps.Clone(s.terrainIdsByType)
	BonusConfigs = append([]TechConfig(nil), s.bonuses...)
	regionNames = append([]string(nil), s.regionNames...)
	lowestIntraMoveCost = s.lowestIntra
	LowestInterMoveCost = s.lowestInter
}
