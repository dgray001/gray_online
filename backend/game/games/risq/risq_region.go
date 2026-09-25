package risq

import (
	"embed"
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/dgray001/gray_online/util"
	"github.com/gin-gonic/gin"
)

//go:embed config/region_names.json
var regionNamesFile embed.FS

var regionNames []string

func init() {
	data, err := regionNamesFile.ReadFile("config/region_names.json")
	if err != nil {
		panic(fmt.Sprintf("failed to read region_names.json: %v", err))
	}
	if err := json.Unmarshal(data, &regionNames); err != nil {
		panic(fmt.Sprintf("failed to parse region_names.json: %v", err))
	}
}

func randomRegionNames(rng *rand.Rand, n int) []string {
	pool := util.ShuffleFrom(rng, append([]string{}, regionNames...))
	if n > len(pool) {
		n = len(pool)
	}
	return pool[:n]
}

type RisqRegion struct {
	name       string
	gold_bonus float64
	spaces     map[uint]bool
	owner      int
}

func (r *GameRisq) addRegion(name string, gold_bonus float64, coordinate_keys map[uint]bool) error {
	for _, existing := range r.regions {
		if existing.name == name {
			return fmt.Errorf("region %q already exists", name)
		}
	}
	for key := range coordinate_keys {
		if invertSpaceKey(key, r) == nil {
			return fmt.Errorf("region %q: invalid space coordinate key %d", name, key)
		}
		for _, existing := range r.regions {
			if existing.spaces[key] {
				return fmt.Errorf("region %q: space %d already belongs to region %q", name, key, existing.name)
			}
		}
	}
	r.regions = append(r.regions, &RisqRegion{name: name, gold_bonus: gold_bonus, spaces: coordinate_keys, owner: -1})
	return nil
}

func (r *GameRisq) regionContaining(space *RisqSpace) *RisqRegion {
	for _, region := range r.regions {
		if region.spaces[space.coordinate_key] {
			return region
		}
	}
	return nil
}

func (r *GameRisq) regionOwner(region *RisqRegion) int {
	owner := -1
	for key := range region.spaces {
		space := invertSpaceKey(key, r)
		if space == nil || space.ownership < 0 {
			return -1
		}
		if owner == -1 {
			owner = space.ownership
		} else if owner != space.ownership {
			return -1
		}
	}
	return owner
}

func (region *RisqRegion) toFrontend(r *GameRisq, player_id int) gin.H {
	explored_keys := make([]uint, 0, len(region.spaces))
	for key := range region.spaces {
		space := invertSpaceKey(key, r)
		if space != nil && space.getVisibility(player_id) >= VisibilityFog {
			explored_keys = append(explored_keys, key)
		}
	}
	if len(explored_keys) == 0 {
		return nil
	}
	return gin.H{
		"name":       region.name,
		"gold_bonus": region.gold_bonus,
		"spaces":     explored_keys,
		"owner":      region.owner,
	}
}

func (r *GameRisq) heldRegions(player_id int) map[string]bool {
	held := make(map[string]bool)
	for _, region := range r.regions {
		if region.owner == player_id {
			held[region.name] = true
		}
	}
	return held
}
