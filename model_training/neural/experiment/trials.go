package experiment

import "fmt"

func trials() []Trial {
	result := []Trial{}
	for _, method := range []string{"hard", "soft"} {
		for _, start := range []string{"zero", "xavier"} {
			for _, rate := range []float64{0.01, 0.05} {
				for _, seed := range []int64{1, 2, 3} {
					result = append(result, Trial{ID: fmt.Sprintf("%s-%s-%.2f-%d", method, start, rate, seed), Method: method, Start: start, Rate: rate, Seed: seed})
				}
			}
		}
	}
	return result
}
