package risq_mapgen_tests

import (
	"hash/fnv"
	"math/rand"
)

// The scripts one per board shape: hexagon, ring, rectangle, triangle
var sweptScripts = []string{"default", "ring", "rectangle", "triangle"}

// 2 and 12 players plus two others from 3 to 11, drawn from a per-script fixed seed so every run tests the same counts
func sweptPlayerCounts(script string) []int {
	hash := fnv.New64a()
	hash.Write([]byte(script))
	others := rand.New(rand.NewSource(int64(hash.Sum64()))).Perm(9)
	return []int{2, 3 + others[0], 3 + others[1], 12}
}
