package rl

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
)

func Run(options Options) error {
	seen := map[int]bool{}
	for _, width := range options.Widths {
		if width < 1 || seen[width] {
			return fmt.Errorf("widths must be distinct positive integers")
		}
		seen[width] = true
	}
	if options.Hidden < 0 || options.CoreShare <= 0 || options.CoreShare >= 1 || math.IsNaN(options.CoreShare) || (options.Hidden > 0 && options.Start == "zero") {
		return fmt.Errorf("invalid architecture or core share")
	}
	if options.GamesPerEpoch < 1 || options.EvaluationGames < 1 || options.TestGames < 1 || options.MaxEpochs < 1 || options.Patience < 1 || options.Workers < 1 || options.Rate <= 0 || math.IsNaN(options.Rate) || math.IsInf(options.Rate, 0) || options.Entropy < 0 || math.IsNaN(options.Entropy) || math.IsInf(options.Entropy, 0) || (options.Start != "zero" && options.Start != "xavier" && options.Start != "small") {
		return fmt.Errorf("invalid RL settings")
	}
	if err := os.MkdirAll(filepath.Dir(options.Output), 0755); err != nil {
		return err
	}
	if err := os.Mkdir(options.Output, 0755); err != nil {
		return err
	}
	validation := schedule(options.EvaluationGames, 1201)
	if err := manifest(options, validation); err != nil {
		return err
	}
	results, err := fitAll(options, validation)
	if err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(options.Output, "trials.json"), results); err != nil {
		return err
	}
	return finish(options, results)
}
