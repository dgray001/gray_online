package rl

import (
	"fmt"
	"strings"
)

func writeTrainingMix(text *strings.Builder, results []Result) {
	games, rounds, decisions, coreGames, coreRounds, coreDecisions := 0, 0, 0, 0, 0, 0
	for _, r := range results {
		for _, e := range r.Curve {
			games += e.Training.Games
			rounds += e.Training.Rounds
			decisions += e.Training.Decisions
			coreGames += e.TrainingCore.Games
			coreRounds += e.TrainingCore.Rounds
			coreDecisions += e.TrainingCore.Decisions
		}
	}
	fmt.Fprintf(text, "## Observed training mix\n\nAcross %d games, %d rounds, and %d nonforced decisions, 4–8 players contributed %.1f%% of games, %.1f%% of rounds, and %.1f%% of decisions.\n\n", games, rounds, decisions, 100*float64(coreGames)/float64(games), 100*float64(coreRounds)/float64(rounds), 100*float64(coreDecisions)/float64(decisions))
}
