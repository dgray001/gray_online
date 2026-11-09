# Fiddlesticks neural network

From `model_training/`:

```bash
go run ./neural/cmd -out neural/results/my-rl-run
go run ./neural/cmd -sweep -out neural/results/reinforce-hidden4
go test ./neural/...
```

New training uses observation schema 2: a 740 → 4 → 79 policy with 3,359
parameters. `-hidden 0` selects a 58,539-parameter linear control.
Existing 370-input checkpoints remain readable and use their original input layout.
See `OBSERVATIONS.md` for public history, player profiles, and game-progress inputs.
Reward is +1 for reaching the bid and 0 for a miss. Fresh games are generated
each epoch; Theory4 supplies opponents, never training targets.
Validation exact-bid success on 4–8 players selects checkpoints; final tests report success,
points, win share, and paired differences against Theory4.
Flags include `-games-per-epoch 400`, `-eval-games 360`, `-test-games 720`,
`-epochs 80`, `-patience 8`, `-rate 0.002`, `-entropy 0.01`, and `-seed 1`.
Training spans 2–10 players, with `-core-share 0.9` targeting 90% of games at
4–8 players. Every game reaches floor(51/players) cards before descending.
`-sweep` compares Xavier/small random starts, two rates, entropy on/off, and
three seeds, plus three fresh linear controls unless `-linear-controls=false`.
`-reference` loads the prior linear checkpoint for matched final evaluation.
Reports separate all players, 4–8 players, upper-half hands, and every hand size.
Checkpoints record categorical bid sampling; old expectation/rounding models remain readable.

Matched single-hidden-layer width sweep:

```bash
go run ./neural/cmd -widths 8,16,32,64,128,256 -sweep -out neural/results/reinforce-width-sweep -games-per-epoch 200 -eval-games 360 -test-games 720 -epochs 160 -patience 16 -workers 20
```

This runs 24 configurations per width and three shared linear controls.
Forward-pass activations are cached for exact round-level backpropagation.
Each finished trial saves its checkpoint and a separate result JSON.

Compare saved checkpoints against native models 0–4 and random-card ablations:

```bash
go run ./neural/cmd -compare-models neural/results/reinforce-width-sweep
```

The comparison reuses saved test games and reports score, success, win share,
and card-policy randomness. It performs no training.

## Historical imitation tools

The following tools and their results are historical imitation experiments.
The old single-run command is now `go run ./neural/imitation/cmd`.
Its flags include `-hidden 32`, `-players 4`, `-max-round 5`, `-rate 0.01`, and `-seed 1`.
Use `-hidden 0` for a direct 370 → 79 linear model with 29,309 parameters.
Scoring flags are `-round-points 10` and `-trick-points 1`, each accepting 0–99.
At least one scoring value must be positive.
The network is a handwritten 370 → 32 → 79 MLP with tanh, masked softmax,
cross-entropy, backpropagation, and stochastic gradient descent. No ML library.

Training imitates Theory4 on generated games: nearest-integer bids and the
highest-weight legal card. Evaluation plays one network seat against Theory4.
Only the player's hand and public state become inputs; illegal actions are masked.
Inputs include player count, round points, and trick points. Scoring is divided by 99.
Models using the previous 368-input layout must be retrained.
JSON files store weights and biases. Training prints loss; evaluation prints scores.
This is an imitation baseline, with no self-play or recorded-human-data loader yet.

## Controlled linear experiment

```bash
go run ./neural/experiment/cmd -out neural/results/linear-baseline -epochs 200
```

Compares hard labels and teacher distributions, zero and Xavier starts,
two learning rates, and three seeds. Games vary 2–6 players, hand limits
2/4/6/8, five scoring presets, and random/Theory2/Theory4/mixed opponents.
Whole-game splits use separate seeds; deals have a separate RNG from actions.
Validation loss controls early stopping and checkpoint retention.
Validation game margins select three configurations for held-out play and a
seat-rotated tournament with Theory4. Each representative uses seed 1.
Outputs include a manifest with source hash, checkpoints, learning curves,
per-game scores, breakdowns by game settings, and `report.md`.
The output directory must not already exist. All training math uses the standard library.

## Compare checkpoint behavior

```bash
go run ./neural/behavior/cmd
```

Evaluates all saved checkpoints on identical situations from 400 fresh games.
The default output is `neural/results/linear-baseline/behavior/`.
`matrix.html` shows pairwise bid and card distances; `report.md` groups changes
by seed, initialization, learning rate, and target type.
`corpus.json` and `fingerprints.json` preserve the situations and predictions.
Use `-corpus PATH -out NEW_DIRECTORY` to reuse the exact saved situation set.
Bid distances reflect expected-bid rounding, matching the engine's actual policy.
