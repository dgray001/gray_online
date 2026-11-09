package experiment

import (
	"fiddlesticks.live/neural"
	"fmt"
	"path/filepath"
)

func fitAndSave(trial Trial, training []neural.Example, validation []neural.Example, test []neural.Example, options Options) (Result, error) {
	result, best := fit(trial, training, validation, options)
	result.Validation = measure(best, validation, trial.Method)
	result.Test = measure(best, test, trial.Method)
	result.Model = filepath.Join(options.Output, trial.ID+".json")
	if err := best.Save(result.Model); err != nil {
		return result, err
	}
	fmt.Printf("finished %s best=%d/%d stop=%s validation=%.5f test=%.5f\n", trial.ID, result.BestEpoch, result.Epochs, result.Stop, result.Validation.Loss, result.Test.Loss)
	return result, nil
}
