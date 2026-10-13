package invariants

import (
	"fmt"
	"testing"
)

func TestDeterminismAndInvariants(t *testing.T) {
	for name, scenario := range map[string]scenario{"economy": economyScenario(), "gathering": gatheringScenario(), "construction": constructionScenario(), "combat": combatScenario(), "eviction": evictionScenario()} {
		t.Run(name, func(t *testing.T) {
			permutations := [][]int{{0, 1}, {1, 0}}
			if scenario.players == 3 {
				permutations = [][]int{{0, 1, 2}, {0, 2, 1}, {1, 0, 2}, {1, 2, 0}, {2, 0, 1}, {2, 1, 0}}
			}
			baseline := runScenario(t, scenario, permutations[0], true)
			configShared.Store(true)
			t.Cleanup(func() { configShared.Store(false) })
			for repeat := range 3 {
				for i, permutation := range permutations {
					if repeat == 0 && i == 0 {
						continue
					}
					t.Run(fmt.Sprintf("run%d/%v", repeat, permutation), func(t *testing.T) {
						t.Parallel()
						compareTrajectory(t, baseline, runScenario(t, scenario, permutation, false))
					})
				}
			}
		})
	}
}
