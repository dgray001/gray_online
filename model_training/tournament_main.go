package main

import (
	"fmt"
	"os"

	"fiddlesticks.live/tournament"
)

func main() {
	// Models in the tournament: Model 3 (baseline champion) vs 4, 4a, 4b, 4c
	models := []tournament.ModelDef{
		{ID: 3, Name: "Model 3", AIModelID: 3, Params: nil},
		{ID: 4, Name: "Model 4", AIModelID: 4, Params: nil},
		{ID: 5, Name: "Model 4a", AIModelID: 5, Params: nil},
		{ID: 6, Name: "Model 4b", AIModelID: 6, Params: nil},
		{ID: 7, Name: "Model 4c", AIModelID: 7, Params: nil},
	}

	// 1. Pentagon Tournament: 5-Player, Modern Scoring (10,3)
	// Every game has all 5 models seated together
	pentagonCfg := tournament.Config{
		Name:        "tournament_pentagon_modern",
		Iterations:  1000,
		MinPlayers:  5,
		MaxPlayers:  5,
		MinRound:    1,
		MaxRound:    10,
		RoundPoints: 10,
		TrickPoints: 3,
		Models:      models,
		OutputDir:   "outputs/tournaments",
	}

	tr := tournament.NewTournamentRunner(pentagonCfg)
	_, err := tr.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Tournament error: %v\n", err)
	}

	// 2. Full Player-Count Sweep (2, 3, 4, 5, 6, 7, 8 Players), Modern Scoring (10,3)
	sweepCfg := tournament.Config{
		Name:        "tournament_sweep_modern",
		Iterations:  300,
		MinPlayers:  2,
		MaxPlayers:  8,
		MinRound:    1,
		MaxRound:    8,
		RoundPoints: 10,
		TrickPoints: 3,
		Models:      models,
		OutputDir:   "outputs/tournaments",
	}

	tr2 := tournament.NewTournamentRunner(sweepCfg)
	_, err = tr2.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Tournament sweep error: %v\n", err)
	}
}
