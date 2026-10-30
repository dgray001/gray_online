# Generalist tactical model: sub-problems and approaches

References are from memory and should be checked before being cited anywhere permanent.

Scope: everything that turns a demand from the strategic layer (see FAMILIES.md) into orders for existing units. Villagers and soldiers share this layer. Gather, build, repair, attack, garrison, scout and evade are all tasks in one assignment problem, and danger is a cost term. Fighting differs from gathering only in being adversarial, which is why combat estimation exists as a shared service.

## 1. Combat outcome estimation

Predict who wins a fight, how long it takes and what survives, from unit stats alone. It is a service used by strategy (the value of power), assignment (assault and retreat odds) and villager safety. Academic roots: Lanchester's laws (1916) as the closed form, and simulation-based estimation (Churchill, Saffidine and Buro, "Fast Heuristic Search for RTS Game Combat Scenarios", AIIDE 2012, with the SparCraft simulator).

### 1a. Analytic exchange model

Compute per-unit damage rates, effective health under armor and penetration, and stamina and range effects directly from `units.json` and tech bonuses, then combine them with a Lanchester-style law over the two mixes. It is cheap enough to run per candidate inside the DSL and extends to any stat change automatically. It ignores positioning, focus fire and garrisons, so it is best as a fast first filter.

### 1b. Playout simulation

Run a simplified fight simulator over the visible armies, with scripted policies for both sides and a fixed number of ticks, and use the outcome distribution. This captures overkill, range and focus effects the analytic model misses. It requires either a Go primitive reusing `ttk/` and `combat.go` or generator-time precomputation of matchup tables keyed on unit and tech state.

## 2. Spatial reasoning

Turn the board into threat, vulnerability, value and opportunity fields so decisions use geometry without hand-written special cases. Academic roots: artificial potential fields (Khatib 1986), their RTS use for movement and kiting (Hagelbäck and Johansson, ORTS), influence maps in game AI (Tozour, *AI Game Programming Wisdom 1*, 2002) and qualitative spatial reasoning for strategy AIs (Forbus, Mahoney and Dill 2002).

### 2a. Threat and value maps

Each space accumulates enemy strength (estimated by 1a, decayed by observation age) and own or enemy asset value, diffused over the zone adjacency graph with terrain costs. Staging points, safe retreat spaces, raid targets and evacuation triggers all become queries on these maps, so they are derived instead of hand-tuned. This extends v7's `estimates.py` and `locations.py` from point estimates to full fields.

### 2b. Reachability and timing analysis

Instead of static strength maps, compute for each space the turn by which each side can bring a given force to it, using movement stamina and path costs. Opportunities and dangers are then defined by arrival-time advantage, which handles fog, seams and speed differences without special rules. It is more expensive but captures timing, which the board's stamina ticks make central.

## 3. Task assignment

Choose which unit performs which task at which target, subject to capacities, travel costs, switching costs and danger. Academic roots: the assignment problem and the Hungarian algorithm (Kuhn 1955), the auction algorithm (Bertsekas 1988) and the multi-robot task allocation taxonomy (Gerkey and Matarić, IJRR 2004).

### 3a. Scored greedy matching

Score every candidate (unit, task, target) with value minus distance, switching and danger terms, then match greedily in rounds with capacities. This is what v7's `assign_nodes` already does for gathering, and the DSL's `assign` with carried rounds supports it natively. It is the least change and generalizes if the score is built from config values and shared prices.

### 3b. Auction or optimal matching

Run an explicit auction or Hungarian-style matching for the hard cases (large armies, many targets) so results do not depend on greedy order. It gives better global assignments and stable results under reassignment. It likely needs a new engine primitive, because the DSL's rounds only approximate it.

## 4. Engagement control and micro

Decide commit, hold, retreat and within-fight actions such as kiting and focus fire. Academic roots: hierarchical adversarial search (Stanescu, Barriga and Buro 2014), portfolio greedy search (Churchill and Buro, AIIDE 2013) and potential-field micro (Hagelbäck and Johansson).

### 4a. Threshold policy over odds

Commit when estimated odds exceed a margin and retreat when they fall below one, with hysteresis. It is simple, cheap and mostly what v7 does (`assault_odds`, `retreat_margin`). With odds coming from 1a or 1b, thresholds become the only tuned parameters and the rest adapts to config.

### 4b. Script portfolio search

Define a small portfolio of scripted behaviors (attack closest, focus weakest, retreat to point, hold), simulate combinations for both sides with 1b, and pick the best response. It handles unusual rosters and stat changes without per-unit rules. It depends on a fast playout and likely needs a Go primitive.

### 4c. Potential-field micro

Each unit moves by summing attraction to targets and repulsion from threats scaled by range and damage, which gives kiting and spacing from stats alone. It is local and robust, but is only worthwhile if the engine's stamina-tick turns leave real room for micro.

## 5. Movement and group coordination

Keep forces cohesive, avoid piecemeal arrivals and move through fog and chokepoints. Academic roots: flocking (Reynolds 1987) and formation and group steering literature.

### 5a. Cohort timing

Group units into cohorts, then gate movement on the slowest member and a staging distance so that arrivals are simultaneous. This is the generalized form of v7's group and march-corridor experiments, using stamina and path cost from the config.

### 5b. Rendezvous planning

Compute for the whole force a meeting space and a departure schedule from the reachability analysis in 2b, then release units on the schedule. It formalizes cohort timing and also covers multi-front pincers, at the cost of a planning step.

## Overall implementations

These combine the pieces above. Each assumes the strategic layer from FAMILIES.md exports demand and that danger is a cost term in assignment.

### I. Incremental generalization of v7

Keep the DSL, rewrite assignment as 3a over all units and tasks, derive thresholds from 1a, and build threat maps with 2a. Replace special rules (`civilian_defense`, `civilian_evacuation`, raid and retreat specifics) with cost terms. This needs no engine changes and is the lowest risk. It gives up the accuracy of playout estimates and optimal matching.

### II. Estimator-centered design

Build 1b as the core: a fast fight simulator available to the AI, with 2b timing, 3a assignment and 4b portfolio search all calling it. Behavior comes mostly from simulated outcomes, so it adapts to arbitrary stat changes. It needs an approved Go primitive and careful step budgets, and is the most future-proof.

### III. Offline-computed tactical tables

Use `ttk/` and `sim/` at generation time to precompute matchup, timing and odds tables keyed on unit and tech state, then have 3a and 4a read them as data. It captures much of II's accuracy without runtime engine changes. Tables go stale if a custom config changes after generation, so it relies on regenerating per config.
