package experiment

import (
	"fmt"
	"os"
	"path/filepath"
)

func Run(options Options) error {
	if options.TrainGames < 1 || options.ValidationGames < 1 || options.TestGames < 1 || options.MaxEpochs < 1 || options.Patience < 1 || options.Workers < 1 {
		return fmt.Errorf("game counts, epochs, patience, and workers must be positive")
	}
	if err := os.MkdirAll(filepath.Dir(options.Output), 0755); err != nil {
		return err
	}
	if err := os.Mkdir(options.Output, 0755); err != nil {
		return err
	}
	training, validation, test := scenarios(options.TrainGames, 11), scenarios(options.ValidationGames, 22), scenarios(options.TestGames, 33)
	if err := manifest(options, training, validation, test); err != nil {
		return err
	}
	data, valid, heldout := collect(training), collect(validation), collect(test)
	fmt.Printf("dataset training=%d validation=%d test=%d nonforced decisions\n", len(data), len(valid), len(heldout))
	results, err := fitAll(options, data, valid, heldout)
	if err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(options.Output, "trials.json"), results); err != nil {
		return err
	}
	return finish(options, results, validation)
}
