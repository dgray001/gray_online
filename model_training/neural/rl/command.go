package rl

import "flag"

func Command() error {
	compare := flag.String("compare-models", "", "compare saved networks with native models 0–4 and random-card baselines")
	refresh := flag.String("refresh-scores", "", "refresh report scores from saved game outcomes")
	output := flag.String("out", "neural/results/reinforce-hidden4", "new output directory")
	games := flag.Int("games-per-epoch", 400, "fresh training games per epoch")
	validation := flag.Int("eval-games", 360, "fixed validation games")
	test := flag.Int("test-games", 720, "held-out games per arena")
	epochs := flag.Int("epochs", 80, "maximum epochs")
	patience := flag.Int("patience", 8, "epochs without validation improvement")
	workers := flag.Int("workers", 4, "parallel trials")
	sweep := flag.Bool("sweep", false, "run the training-configuration sweep")
	seed := flag.Int64("seed", 1, "training seed")
	rate := flag.Float64("rate", 0.002, "policy-gradient learning rate")
	entropy := flag.Float64("entropy", 0.01, "entropy regularization")
	start := flag.String("start", "xavier", "xavier, small, or zero for a linear model")
	hidden := flag.Int("hidden", 4, "hidden neurons; zero selects a linear policy")
	widths := flag.String("widths", "", "comma-separated widths for a matched sweep")
	core := flag.Float64("core-share", 0.9, "share of training games with 4–8 players")
	controls := flag.Bool("linear-controls", true, "include three linear controls in a hidden-layer sweep")
	reference := flag.String("reference", "neural/results/reinforce-baseline/rl-xavier-r0.0020-e0.00-s1.json", "previous linear checkpoint; empty disables")
	reference4 := flag.String("reference-hidden4", "neural/results/reinforce-hidden4/rl-small-r0.0020-e0.00-h4-s1.json", "previous four-neuron checkpoint; empty disables")
	flag.Parse()
	if *compare != "" {
		return CompareModels(*compare)
	}
	if *refresh != "" {
		return RefreshScores(*refresh)
	}
	parsed, err := parseWidths(*widths)
	if err != nil {
		return err
	}
	return Run(Options{Output: *output, GamesPerEpoch: *games, EvaluationGames: *validation, TestGames: *test, MaxEpochs: *epochs, Patience: *patience, Workers: *workers, Sweep: *sweep, Seed: *seed, Rate: *rate, Entropy: *entropy, Start: *start, Hidden: *hidden, CoreShare: *core, LinearControls: *controls, Reference: *reference, Widths: parsed, ReferenceHidden4: *reference4})
}
