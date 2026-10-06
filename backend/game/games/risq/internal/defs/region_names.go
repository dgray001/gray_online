package defs

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"slices"

	"github.com/dgray001/gray_online/util"
)

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
