package risq

import (
	"os"
	"path/filepath"
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
		data, err := readConfigFile(l.file)
		if err != nil {
			return err
		}
		l.load(data)
	}
	return nil
}

func readConfigFile(parts ...string) ([]byte, error) {
	return os.ReadFile(filepath.Join(append([]string{configDir}, parts...)...))
}
