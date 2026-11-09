package experiment

import "fiddlesticks.live/neural"

type StandardSet struct {
	Seed       int64
	Games      []Scenario
	Situations []neural.Example
}

func StandardSituations(games int, seed int64) StandardSet {
	schedule := scenarios(games, seed)
	return StandardSet{Seed: seed, Games: schedule, Situations: collect(schedule)}
}
