package experiment

import (
	"math"
	"slices"
)

func finishRegimens(groups map[string]*Regimen) []Regimen {
	result := []Regimen{}
	for _, g := range groups {
		n := float64(g.Seeds)
		g.MarginSD = math.Sqrt(math.Max(0, (g.MarginSD-g.Margin*g.Margin/n)/math.Max(1, n-1)))
		g.Margin /= n
		g.WinShare /= n
		g.TeacherKL /= n
		g.Accuracy /= n
		result = append(result, *g)
	}
	slices.SortFunc(result, func(a Regimen, b Regimen) int {
		if a.Margin > b.Margin {
			return -1
		}
		if a.Margin < b.Margin {
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
