# Outcome-based reinforcement learning

From `model_training/`, run `go run ./neural/cmd` or add `-sweep` for 24 hidden-layer trials and three linear controls.
The default policy has four hidden tanh neurons and 1,879 parameters.
Hidden policies start from Xavier or smaller random weights; all training starts fresh.

The network samples a legal bid and legal cards from categorical probabilities.
At round completion it receives 1 for reaching the bid and 0 for missing.
That reward credits all sampled nonforced decisions in the round using REINFORCE.
A running baseline by player count and round size reduces variance.
Gradients use the probabilities recorded when the actions were sampled.
An entropy bonus can encourage exploration. Network math is handwritten Go.

Each epoch generates fresh full-arc games for 2–10 players, targeting 90% of games at 4–8 players.
Maximum hands are 25/17/12/10/8/7/6/5/5 cards for 2–10 players.
Validation replays balanced independent deals; 4–8-player success controls checkpointing.
Frozen round weights supply the complete multi-action hidden-layer gradient.
Training includes uniform, Theory2, Theory4, mixed, and frozen past-self opponents.
Theory4 supplies no targets or gradients.

Final tests measure exact-bid success, points, win share, and zero-bid frequency.
Binary exact-bid reward and game-winning reward are different objectives.
The report includes paired whole-game bootstrap intervals against Theory4.
The previous short-round 800-game benchmark is also replayed for direct comparison.
