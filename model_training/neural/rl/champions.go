package rl

func champions(results []Result, ranking []Regimen) []Result {
	selected := []Result{}
	counts := map[int]int{}
	for _, regimen := range ranking {
		if counts[regimen.Hidden] >= 3 {
			continue
		}
		candidate := -1
		for i, result := range results {
			if regimenID(result.Trial) == regimen.ID && (candidate < 0 || result.Trial.Seed == 1) {
				candidate = i
			}
		}
		if candidate >= 0 {
			selected = append(selected, results[candidate])
			counts[regimen.Hidden]++
		}
	}
	return selected
}
