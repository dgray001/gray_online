package defs

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dgray001/gray_online/util"
	"io/fs"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"slices"
)

// Set once by LoadConfig; ai configs and map scripts are read from under it when a game is created
var configDir string

var configFiles = []struct {
	file string
	load func(c *Config, data []byte)
}{
	{"bonuses.json", (*Config).loadBonuses},
	{"buildings.json", (*Config).loadBuildings},
	{"region_names.json", (*Config).loadRegionNames},
	{"resources.json", (*Config).loadResources},
	{"techs.json", (*Config).loadTechs},
	{"terrains.json", (*Config).loadTerrains},
	{"units.json", (*Config).loadUnits},
}

// Every binary calls this once at startup, before creating any game, so config edits never need a rebuild
func LoadConfig(dir string) error {
	configDir = dir
	c, err := parseConfig("")
	if err != nil {
		return err
	}
	c.installAsDefault()
	return nil
}

// Replaces the default config's units with those parsed from an arbitrary units json file
func LoadUnitsFile(path string) (err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("units file %q: %v", path, recovered)
		}
	}()
	c := DefaultConfig()
	c.loadUnits(data)
	c.installAsDefault()
	return nil
}

// A game's own Config: each file comes from the override folder under the config dir when it has one, else from the config dir itself
func LoadConfigDir(override string) (config *Config, err error) {
	if !overrideNamePattern.MatchString(override) {
		return nil, fmt.Errorf("config dir %q must be lowercase letters, digits and underscores", override)
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			config, err = nil, fmt.Errorf("config dir %q: %v", override, recovered)
		}
	}()
	return parseConfig(override)
}

var overrideNamePattern = regexp.MustCompile(`^[a-z0-9_]+$`)

func parseConfig(override string) (*Config, error) {
	c := newConfig()
	for _, f := range configFiles {
		data, err := ReadConfigFile(override, f.file)
		if override != "" && errors.Is(err, fs.ErrNotExist) {
			data, err = ReadConfigFile(f.file)
		}
		if err != nil {
			return nil, err
		}
		f.load(c, data)
	}
	return c, nil
}

func configPath(parts ...string) (string, error) {
	rel := filepath.Join(parts...)
	if rel != "" && !filepath.IsLocal(rel) {
		return "", fmt.Errorf("config path %q escapes the config dir", rel)
	}
	return filepath.Join(configDir, rel), nil
}

func ReadConfigFile(parts ...string) ([]byte, error) {
	path, err := configPath(parts...)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(path)
}

func WriteConfigFile(data []byte, parts ...string) error {
	path, err := configPath(parts...)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func RemoveConfigFile(parts ...string) error {
	path, err := configPath(parts...)
	if err != nil {
		return err
	}
	return os.Remove(path)
}

func ListConfigDir(parts ...string) ([]os.DirEntry, error) {
	path, err := configPath(parts...)
	if err != nil {
		return nil, err
	}
	return os.ReadDir(path)
}

var regionNames []string

func (c *Config) loadRegionNames(data []byte) {
	if err := json.Unmarshal(data, &c.RegionNames); err != nil {
		panic(fmt.Sprintf("failed to parse region_names.json: %v", err))
	}
}

// Up to n distinct names from the pool, never any in exclude
func (c *Config) RandomRegionNames(rng *rand.Rand, n int, exclude []string) []string {
	pool := make([]string, 0, len(c.RegionNames))
	for _, name := range c.RegionNames {
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
