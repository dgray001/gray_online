package main

import (
	"fiddlesticks.live/neural/experiment"
	"flag"
	"fmt"
	"os"
)

func main() {
	output := flag.String("out", "neural/results/linear-baseline", "new output directory")
	training := flag.Int("train-games", 800, "training games")
	validation := flag.Int("validation-games", 400, "validation games")
	test := flag.Int("test-games", 400, "test and tournament games")
	epochs := flag.Int("epochs", 80, "maximum epochs")
	patience := flag.Int("patience", 8, "epochs without meaningful improvement")
	workers := flag.Int("workers", 4, "parallel training jobs")
	flag.Parse()
	err := experiment.Run(experiment.Options{Output: *output, TrainGames: *training, ValidationGames: *validation, TestGames: *test, MaxEpochs: *epochs, Patience: *patience, Workers: *workers})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
