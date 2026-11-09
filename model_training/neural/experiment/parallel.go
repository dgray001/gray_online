package experiment

import (
	"fiddlesticks.live/neural"
	"sync"
)

func fitAll(options Options, training []neural.Example, validation []neural.Example, test []neural.Example) ([]Result, error) {
	configs := trials()
	results, failures := make([]Result, len(configs)), make(chan error, len(configs))
	limit, wait := make(chan struct{}, options.Workers), sync.WaitGroup{}
	for i, config := range configs {
		limit <- struct{}{}
		wait.Add(1)
		go func() {
			defer wait.Done()
			defer func() { <-limit }()
			var err error
			results[i], err = fitAndSave(config, training, validation, test, options)
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
