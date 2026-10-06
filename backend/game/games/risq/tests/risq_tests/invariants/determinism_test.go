package invariants

import (
	"fmt"
	"testing"
)

func TestDeterminismAndInvariants(t *testing.T) {
	for name, scenario := range map[string]scenario{"economy": economyScenario(), "gathering": gatheringScenario(), "construction": constructionScenario(), "combat": combatScenario(), "production": productionScenario(), "farm": farmScenario(), "eviction": evictionScenario(), "capture": captureScenario(), "budget": budgetScenario()} {
		t.Run(name, func(t *testing.T) {
			permutations := [][]int{{0, 1}, {1, 0}}
			if scenario.players == 3 {
				permutations = [][]int{{0, 1, 2}, {0, 2, 1}, {1, 0, 2}, {1, 2, 0}, {2, 0, 1}, {2, 1, 0}}
			}
			var baseline []string
			for repeat := range 3 {
				for _, permutation := range permutations {
					t.Run(fmt.Sprintf("run%d/%v", repeat, permutation), func(t *testing.T) {
						got := runScenario(t, scenario, permutation)
						if baseline == nil {
							baseline = got
						} else {
							compareTrajectory(t, baseline, got)
						}
					})
				}
			}
		})
	}
}
