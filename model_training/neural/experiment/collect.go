package experiment

import (
	"fiddlesticks.live/neural"
	"fmt"
	"math/rand"
)

func collect(schedule []Scenario) []neural.Example {
	result := []neural.Example{}
	for i, scenario := range schedule {
		rand.Seed(scenario.ActionSeed)
		examples := []neural.Example{}
		neural.Play(nil, &examples, scenario.Settings, scenario.Seat)
		for _, example := range examples {
			if len(example.Legal) > 1 {
				result = append(result, example)
			}
		}
		if (i+1)%100 == 0 {
			fmt.Printf("collected %d/%d games, %d decisions\n", i+1, len(schedule), len(result))
		}
	}
	return result
}
