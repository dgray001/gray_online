package defs

import (
	"encoding/json"
	"fmt"
	"github.com/dgray001/gray_online/util"
	"math/rand"
	"os"
	"path/filepath"
	"slices"
)

// Set once by LoadConfig; ai configs and map scripts are read from under it when a game is created
var configDir string

// Every binary calls this once at startup, before creating any game, so config edits never need a rebuild
func LoadConfig(dir string) error {
	configDir = dir
	loaders := []struct {
		file string
		load func([]byte)
	}{
		{"bonuses.json", loadBonusConfig},
		{"buildings.json", loadBuildingConfig},
		{"region_names.json", loadRegionNames},
		{"resources.json", loadResourceConfig},
		{"techs.json", loadTechConfig},
		{"terrains.json", loadTerrainConfig},
		{"units.json", loadUnitConfig},
	}
	for _, l := range loaders {
		data, err := ReadConfigFile(l.file)
		if err != nil {
			return err
		}
		l.load(data)
	}
	return nil
}

func ReadConfigFile(parts ...string) ([]byte, error) {
	return os.ReadFile(filepath.Join(append([]string{configDir}, parts...)...))
}

func WriteConfigFile(data []byte, parts ...string) error {
	return os.WriteFile(filepath.Join(append([]string{configDir}, parts...)...), data, 0o644)
}

func RemoveConfigFile(parts ...string) error {
	return os.Remove(filepath.Join(append([]string{configDir}, parts...)...))
}

func ListConfigDir(parts ...string) ([]os.DirEntry, error) {
	return os.ReadDir(filepath.Join(append([]string{configDir}, parts...)...))
}

var regionNames []string

func loadRegionNames(data []byte) {
	if err := json.Unmarshal(data, &regionNames); err != nil {
		panic(fmt.Sprintf("failed to parse region_names.json: %v", err))
	}
}

// Up to n distinct names from the pool, never any in exclude
func RandomRegionNames(rng *rand.Rand, n int, exclude []string) []string {
	pool := make([]string, 0, len(regionNames))
	for _, name := range regionNames {
		if !slices.Contains(exclude, name) {
			pool = append(pool, name)
		}
	}
	util.ShuffleFrom(rng, pool)
	if n > len(pool) {
		n = len(pool)
	}
	return pool[:n]
}
