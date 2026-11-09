package rl

import (
	"math"
	"slices"
)

func finishRegimens(groups map[string]*Regimen) []Regimen {
	result := []Regimen{}
	for _, g := range groups {
		n := float64(g.Seeds)
		g.ExactSD = math.Sqrt(math.Max(0, (g.ExactSD-g.Exact*g.Exact/n)/math.Max(1, n-1)))
		g.Initial /= n
		g.Exact /= n
		g.WinShare /= n
		g.Margin /= n
		result = append(result, *g)
	}
	slices.SortFunc(result, func(a Regimen, b Regimen) int {
		if a.Exact > b.Exact {
			return -1
		}
		if a.Exact < b.Exact {
			return 1
		}
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
	return result
}
