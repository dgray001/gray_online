package rl

import (
	"fiddlesticks.live/neural/experiment"
	"path/filepath"
	"runtime"
)

func manifest(options Options, validation []experiment.Scenario) error {
	source, err := experiment.SourceHash()
	if err != nil {
		return err
	}
	value := struct {
		Options                                    Options
		Trials                                     []Trial
		Algorithm, Reward, BidPolicy, TrainingRule string
		Validation, MixedTest, Theory4Test         []experiment.Scenario
		SourceSHA256, GoVersion                    string
	}{options, trials(options), "REINFORCE with action-independent baseline", "1 for exact bid, 0 for miss; no teacher targets", "categorical", "fresh full-arc 2–10 player schedule seed 100000+epoch, weighted by CoreShare; action seed += trial seed * 1e12; every fifth training game uses a frozen start-of-epoch opponent; selection prioritizes 4–8 players", validation, schedule(options.TestGames, 1601), directSchedule(options.TestGames, 1602), source, runtime.Version()}
	return writeJSON(filepath.Join(options.Output, "manifest.json"), value)
}
