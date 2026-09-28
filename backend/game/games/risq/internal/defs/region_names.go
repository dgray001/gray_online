package defs

import (
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/dgray001/gray_online/util"
)

var regionNames []string

func loadRegionNames(data []byte) {
	if err := json.Unmarshal(data, &regionNames); err != nil {
		panic(fmt.Sprintf("failed to parse region_names.json: %v", err))
	}
}

func RandomRegionNames(rng *rand.Rand, n int) []string {
	pool := util.ShuffleFrom(rng, append([]string{}, regionNames...))
	if n > len(pool) {
		n = len(pool)
	}
	return pool[:n]
}
