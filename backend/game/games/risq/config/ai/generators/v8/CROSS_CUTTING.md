# Generalist cross-cutting concerns: objective, uncertainty and capability

References are from memory and should be checked before being cited anywhere permanent.

Scope: the pieces that FAMILIES.md (strategic) and TACTICS.md (execution) both depend on and that neither defines. Expansion, plan commitment and multiplayer targeting are not listed because they reduce to action rows, cost terms and per-opponent beliefs inside these documents.

## 1. Objective function

Every strategic and tactical score needs one definition of "good" derived from the rules, so power, economy, survival and territory convert into progress toward winning. It is what gives the value ledger a common unit and lets strategy and tactics trade off against each other. Academic roots: utility theory and reward definition in MDPs, potential-based reward shaping for converting intermediate quantities into terminal progress without changing optimal behavior (Ng, Harada and Russell 1999), and evaluation-function learning in games (Samuel 1959; Tesauro's TD-Gammon 1995).

### 1a. Rule-derived progress measure

Define value as estimated distance to victory from the rules: progress toward the opponent's elimination condition (no units and no unit-producing building), own survival margin, and the `simoutcome` tiebreakers of unit ratio, score and land at the turn limit. Intermediate quantities convert through the same estimator as tactics (1 in TACTICS.md), so a unit of economy is worth what it converts into over the horizon. It needs no training and adapts to config changes because it reads the rules, but the conversion rates are modeled, not measured.

### 1b. Fitted outcome predictor

Regress the final result (win probability) on state features such as power ratio, economy flow, land, tech and production capacity, using games from `sim/`. The fitted function supplies the conversion rates between features and can be refit per config. It is measured instead of assumed, but is only as good as the games and opponents in the training set and needs refitting when configs change. A reasonable use is to calibrate 1a's conversion rates instead of replacing it.

## 2. Uncertainty and opponents

Fog hides enemy units, buildings and economy, so decisions are made on a belief, not on the state. This one problem covers scouting (the value of reducing uncertainty), multiple opponents (a belief per opponent plus target choice), robustness to noisy estimates (how the belief is used) and strategic posture (the output of the value model given the objective and the belief about relative power and economy). Academic roots: partially observable Markov decision processes (Kaelbling, Littman and Cassandra 1998), RTS state estimation by particle filter (Weber, Mateas and Jhala, AIIDE 2011), value of information (Howard 1966) and imperfect-information search (Ginsberg's GIB 2001; information-set MCTS, Cowling, Powley and Whitehouse 2012).

### 2a. Point estimate with age and confidence

Keep the best estimate of each opponent's army, economy and production, with the time since last observation, and widen the estimate as it ages. Decisions use a risk-adjusted value (for example the estimate scaled toward the worst plausible case when stakes are high). This extends v7's `estimates.py` and unidentified-space sums and is cheap enough for the DSL. It does not capture correlated uncertainty.

### 2b. Sampled hidden states

Sample several plausible hidden states consistent with observations (from a prior over build progress and unit mixes), evaluate each candidate decision against every sample, and choose the decision with the best average or risk-adjusted value. It gives robust decisions and a direct measure of how much a decision depends on the unknowns. It requires evaluation to be fast and likely needs a Go primitive or generation-time precomputation.

### 2c. Information value for scouting

Score each scouting action by how much its expected observation would change the best decision, using the spread from 2a or 2b, minus its cost and risk. Scouting then enters the ledger as an action row instead of running as a separate module (`scouting.py` in v7). It makes scouting adapt to how uncertain and how consequential a decision is.

## 3. Engine and DSL capability

This is a decision gate, not a design area. Several approaches need runtime support the AI DSL may not provide: forward simulation (FAMILIES.md 2a), optimal matching (TACTICS.md 3b), playout simulation (TACTICS.md 1b and 4b) and sampled evaluation (2b above). Each rule is also limited to 250,000 select steps and 10,000 loop steps per decision.

Before choosing a family, inventory each approach by how it can run:

1. **DSL only:** expressions, selectors and `assign` rounds as they exist today.
2. **Generation time:** the Python generator computes tables or priors from the config and `sim/`, and emits them as data.
3. **New Go primitive:** a general engine capability added with your approval, such as a fight-playout function.

The inventory decides which of the strategic and tactical implementations are available without engine changes. It also identifies which Go primitives would unlock the most approaches for the least code, so the engine work, if any, can be requested once and kept general.
