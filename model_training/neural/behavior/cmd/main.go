package main

import (
	"fiddlesticks.live/neural/behavior"
	"flag"
	"fmt"
	"os"
)

func main() {
	run := flag.String("run", "neural/results/linear-baseline", "experiment directory")
	out := flag.String("out", "neural/results/linear-baseline/behavior", "new output directory")
	corpus := flag.String("corpus", "", "reuse a saved corpus.json")
	games := flag.Int("games", 400, "standard games")
	seed := flag.Int64("seed", 55, "standard-game seed")
	flag.Parse()
	if err := behavior.Run(behavior.Options{Run: *run, Output: *out, Corpus: *corpus, Games: *games, Seed: *seed}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
