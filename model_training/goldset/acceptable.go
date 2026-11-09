package goldset

import (
	"fmt"
	"slices"
)

func acceptableActions(item Item, primary int, actions []int) ([]int, error) {
	if len(actions) == 0 {
		actions = []int{primary}
	}
	result := slices.Clone(actions)
	slices.Sort(result)
	result = slices.Compact(result)
	if !slices.Contains(result, primary) {
		return nil, fmt.Errorf("original choice must be an acceptable action")
	}
	for _, action := range result {
		if !slices.Contains(item.Legal, action) {
			return nil, fmt.Errorf("all acceptable actions must be legal")
		}
	}
	return result, nil
}
