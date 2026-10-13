package risq_mapgen_tests

import (
	"hash/fnv"
	"math/rand"
)

// The scripts one per board shape: hexagon, ring, rectangle, triangle
var sweptScripts = []string{"hexagon", "ring", "rectangle", "triangle"}

// One count from 3 to 11 drawn from a per-script fixed seed, so every run tests the same counts, then the 12 player maximum
func sweptPlayerCounts(script string) []int {
	hash := fnv.New64a()
	hash.Write([]byte(script))
	others := rand.New(rand.NewSource(int64(hash.Sum64()))).Perm(9)
	return []int{3 + others[0], 12}
}

// Two of the four scripts, rotating with n, for checks that don't depend on a board's shape enough to need all four
func scriptPair(n int) []string {
	return []string{sweptScripts[n%4], sweptScripts[(n+1)%4]}
}
