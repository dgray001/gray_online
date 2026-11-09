package experiment

import (
	"fiddlesticks.live/neural"
	"fmt"
)

func evaluate(results []Result, schedule []Scenario) ([]GameResult, []Evaluation, error) {
	rows, summaries := []GameResult{}, []Evaluation{}
	for _, result := range results {
		network, err := neural.Load(result.Model)
		if err != nil {
			return nil, nil, err
		}
		games, summary := arena(result.Trial.ID, network, schedule)
		rows, summaries = append(rows, games...), append(summaries, summary...)
		fmt.Printf("evaluated %s games=%d\n", result.Trial.ID, len(schedule))
	}
	return rows, summaries, nil
}
