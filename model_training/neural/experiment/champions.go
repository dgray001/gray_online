package experiment

func champions(results []Result, ranking []Regimen) []Result {
	selected := []Result{}
	for _, regimen := range ranking[:min(3, len(ranking))] {
		for _, result := range results {
			if regimenID(result.Trial) == regimen.ID && result.Trial.Seed == 1 {
				selected = append(selected, result)
			}
		}
	}
	return selected
}
