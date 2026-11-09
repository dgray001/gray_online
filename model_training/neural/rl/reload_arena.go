package rl

import (
	"fmt"
	"path/filepath"
)

func reloadArena(directory string, filename string, previous []Evaluation) ([]Evaluation, error) {
	if len(previous) == 0 {
		return previous, nil
	}
	var games map[string][]Episode
	if err := readJSON(filepath.Join(directory, filename), &games); err != nil {
		return nil, err
	}
	result := make([]Evaluation, len(previous))
	for i, evaluation := range previous {
		episodes := games[evaluation.Name]
		if len(episodes) == 0 {
			return nil, fmt.Errorf("missing game records for %s", evaluation.Name)
		}
		result[i] = summarize(evaluation.Name, episodes)
	}
	return result, nil
}
