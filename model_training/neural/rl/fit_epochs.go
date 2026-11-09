package rl

import (
	"fiddlesticks.live/neural/experiment"
	"fmt"
	"path/filepath"
)

func (f *fitting) run(options Options, scenarios []experiment.Scenario) (Result, error) {
	for epoch := 1; epoch <= options.MaxEpochs; epoch++ {
		training := trainEpoch(f.network, f.result.Trial, f.learner, epoch, options)
		validation, _ := evaluate(f.result.Trial.ID, f.network, scenarios)
		f.result.Epochs = epoch
		f.result.Curve = append(f.result.Curve, Epoch{Number: epoch, Training: training.All, Validation: validation.All, TrainingCore: selectionMetrics(training), ValidationCore: selectionMetrics(validation)})
		if f.consider(validation, epoch) {
			if err := f.best.Save(f.result.Model); err != nil {
				return f.result, err
			}
		}
		if epoch%5 == 0 {
			fmt.Printf("%s epoch=%d core train=%.3f validation=%.3f best=%.3f\n", f.result.Trial.ID, epoch, selectionMetrics(training).ExactRate, selectionMetrics(validation).ExactRate, selectionMetrics(f.result.Validation).ExactRate)
		}
		if err := writeJSON(filepath.Join(options.Output, f.result.Trial.ID+"-progress.json"), f.result); err != nil {
			return f.result, err
		}
		if f.stale >= options.Patience {
			f.result.Stop = "validation plateau"
			break
		}
	}
	fmt.Printf("finished %s best=%d/%d core exact=%.4f stop=%s\n", f.result.Trial.ID, f.result.BestEpoch, f.result.Epochs, selectionMetrics(f.result.Validation).ExactRate, f.result.Stop)
	return f.result, nil
}
