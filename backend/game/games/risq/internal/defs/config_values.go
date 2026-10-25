package defs

import (
	"fmt"
	"maps"
)

type fieldTable[T any] map[string]func(T) float64

type configTable struct {
	has   func(field string) bool
	value func(c *Config, field string, id uint32) (float64, error)
}

func newConfigTable[T any](name string, fields fieldTable[T], entry func(*Config, uint32) (T, bool)) configTable {
	return configTable{
		has: func(field string) bool { return fields[field] != nil },
		value: func(c *Config, field string, id uint32) (float64, error) {
			get := fields[field]
			if get == nil {
				return 0, fmt.Errorf("config(%s, %s, %d): unknown field", name, field, id)
			}
			e, found := entry(c, id)
			if !found {
				return 0, nil
			}
			return get(e), nil
		},
	}
}

func entriesOf[T any](pick func(*Config) map[uint32]T) func(*Config, uint32) (T, bool) {
	return func(c *Config, id uint32) (T, bool) {
		e, ok := pick(c)[id]
		return e, ok
	}
}

var configTables map[string]configTable

func init() {
	configTables = map[string]configTable{
		"unit":     newConfigTable("unit", unitFields, entriesOf(func(c *Config) map[uint32]UnitConfig { return c.Units })),
		"building": newConfigTable("building", buildingFields, entriesOf(func(c *Config) map[uint32]BuildingConfig { return c.Buildings })),
		"tech":     newConfigTable("tech", techFields, entriesOf(func(c *Config) map[uint32]TechConfig { return c.Techs })),
		"resource": newConfigTable("resource", resourceFields, entriesOf(func(c *Config) map[uint32]ResourceConfig { return c.Resources })),
		"terrain":  newConfigTable("terrain", terrainFields, terrainEntry),
	}
}

func HasConfigField(table string, field string) bool {
	t, ok := configTables[table]
	return ok && t.has(field)
}

func (c *Config) ConfigValue(table string, field string, id uint32) (float64, error) {
	t, ok := configTables[table]
	if !ok {
		return 0, fmt.Errorf("config(%s, %s, %d): unknown table", table, field, id)
	}
	return t.value(c, field, id)
}

func merge[T any](tables ...fieldTable[T]) fieldTable[T] {
	merged := fieldTable[T]{}
	for _, t := range tables {
		maps.Copy(merged, t)
	}
	return merged
}

func costFields[T any](prefix string, cost func(T) RisqResourceCost) fieldTable[T] {
	return fieldTable[T]{
		prefix + "food":  func(e T) float64 { return cost(e).Food },
		prefix + "wood":  func(e T) float64 { return cost(e).Wood },
		prefix + "stone": func(e T) float64 { return cost(e).Stone },
		prefix + "gold":  func(e T) float64 { return cost(e).Gold },
	}
}

func visionFields[T any](vision func(T) RisqVision) fieldTable[T] {
	return fieldTable[T]{
		"vision_space":         func(e T) float64 { return float64(vision(e).Space) },
		"vision_edge_adjacent": func(e T) float64 { return float64(vision(e).Edge_adjacent) },
		"vision_adjacent":      func(e T) float64 { return float64(vision(e).Adjacent) },
		"vision_edge_opposite": func(e T) float64 { return float64(vision(e).Edge_opposite) },
		"vision_secondary":     func(e T) float64 { return float64(vision(e).Secondary) },
	}
}

func boolValue(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
