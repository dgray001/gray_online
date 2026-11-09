package rl

import (
	"fiddlesticks.live/neural/experiment"
	"path/filepath"
	"sync"
)

func fitAll(options Options, validation []experiment.Scenario) ([]Result, error) {
	configs := trials(options)
	results := make([]Result, len(configs))
	limit, failures, wait := make(chan struct{}, options.Workers), make(chan error, len(configs)), sync.WaitGroup{}
	for i, config := range configs {
		limit <- struct{}{}
		wait.Add(1)
		go func() {
			defer wait.Done()
			defer func() { <-limit }()
			var err error
			results[i], err = fit(config, options, validation)
			if err == nil {
				err = writeJSON(filepath.Join(options.Output, config.ID+"-result.json"), results[i])
			}
			if err != nil {
				failures <- err
			}
		}()
	}
	wait.Wait()
	close(failures)
	for err := range failures {
		return nil, err
	}
	return results, nil
}
