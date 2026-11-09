package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"

	"fiddlesticks.live/neural"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	games := flag.Int("games", 100, "legacy imitation games to collect")
	epochs := flag.Int("epochs", 10, "training passes")
	hidden := flag.Int("hidden", 32, "hidden neurons; zero selects a linear model")
	players := flag.Int("players", 4, "players per game")
	maxRound := flag.Int("max-round", 5, "largest hand size")
	roundPoints := flag.Int("round-points", 10, "points for an exact bid")
	trickPoints := flag.Int("trick-points", 1, "additional points per bid trick")
	rate := flag.Float64("rate", 0.01, "learning rate")
	seed := flag.Int64("seed", 1, "random seed")
	model := flag.String("model", "network.json", "output model path")
	load := flag.String("load", "", "existing model path")
	evaluate := flag.Int("eval", 20, "evaluation games against Theory4")
	flag.Parse()
	if *maxRound < 1 || *maxRound > 25 || *roundPoints < 0 || *roundPoints > 99 || *trickPoints < 0 || *trickPoints > 99 || *roundPoints+*trickPoints == 0 {
		return fmt.Errorf("require max-round 1–25 and scoring values 0–99 with at least one positive")
	}
	settings := neural.GameSettings{Players: *players, MaxRound: uint8(*maxRound), RoundPoints: uint16(*roundPoints), TrickPoints: uint16(*trickPoints)}
	return execute(*games, *epochs, *hidden, settings, *rate, *seed, *model, *load, *evaluate)
}

func execute(games int, epochs int, hidden int, settings neural.GameSettings, rate float64, seed int64, model string, load string, evaluate int) error {
	if games < 0 || epochs < 1 || hidden < 0 || settings.Players < 2 || settings.Players > 51 || int(settings.MaxRound)*settings.Players > 51 || rate <= 0 || evaluate < 1 {
		return fmt.Errorf("invalid settings: require positive epochs, hidden size and rate, 2–51 players, and max-round * players <= 51")
	}
	rand.Seed(seed)
	network := neural.NewNetwork(neural.Inputs, hidden, neural.Outputs, seed)
	if load != "" {
		var err error
		network, err = neural.Load(load)
		if err != nil {
			return err
		}
		if network.InputCount() != neural.Inputs || len(network.OutputBias) != neural.Outputs {
			return fmt.Errorf("model does not match the Fiddlesticks input/output dimensions")
		}
	}
	examples := []neural.Example{}
	for i := 0; i < games; i++ {
		neural.Play(network, &examples, settings, 0)
	}
	fmt.Printf("collected %d decisions from %d games\n", len(examples), games)
	train(network, examples, epochs, rate, seed)
	if games > 0 {
		if err := network.Save(model); err != nil {
			return err
		}
	}
	benchmark(network, settings, evaluate)
	return nil
}

func train(network *neural.Network, examples []neural.Example, epochs int, rate float64, seed int64) {
	if len(examples) == 0 {
		return
	}
	rng := rand.New(rand.NewSource(seed))
	for epoch := 0; epoch < epochs; epoch++ {
		rng.Shuffle(len(examples), func(i int, j int) { examples[i], examples[j] = examples[j], examples[i] })
		loss := 0.0
		for _, example := range examples {
			loss += network.Train(example.Input, example.Legal, example.Target, rate)
		}
		fmt.Printf("epoch %d loss %.4f\n", epoch+1, loss/float64(len(examples)))
	}
}
