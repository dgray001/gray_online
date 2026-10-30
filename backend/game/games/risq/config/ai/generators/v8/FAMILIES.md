# Generalist demand model: families of approach

References are from memory and should be checked before being cited anywhere permanent.

## 1. Value and price-based allocation

Every possible action is scored on one common scale (expected benefit minus the cost of the scarce resources it consumes), and spending follows the scores. Resources carry a price that reflects what the best blocked purchase would earn, so scarcity and priority emerge instead of being prescribed. Academic roots: marginal analysis and shadow prices from microeconomics and constrained optimization (Lagrange multipliers), utility theory in game AI (Dave Mark, *Behavioral Mathematics for Game AI*, 2009; the Infinite Axis Utility System, GDC 2013), and market-oriented programming for distributed resource allocation (Wellman, JAIR 1993).

### 1a. Marginal-value ledger

A generic enumerator turns every unit, building and tech in the config into a row holding a cost vector, delivery delay and deltas to flow, power and capacity. Each row is scored by discounted net benefit over a finite horizon, and the best affordable row is bought each iteration. When an unaffordable row beats every affordable one, the AI saves for it and holds the resource price. Tech value is the affected units (owned plus projected production) times the stat delta converted into power. Prerequisites inherit the discounted value of what they unlock.

### 1b. Auction of bidders

Each entity kind (producer slot, unit class, tech, builder crew, expansion site, defense) is a self-contained bidder that submits a value per unit of delay avoided and a cost vector. Resources clear at a per-turn price: bids are funded in descending value per scarcity-weighted cost until stock plus expected flow runs out. The clearing prices also drive worker allocation. This is the same math as 1a with a different decomposition, and its advantage is extensibility and inspectability ("who bid what, who won").

## 2. Search and simulation

Instead of scoring actions by formula, the AI simulates the consequences of candidate plans in an abstract model and picks the plan with the best simulated outcome. Plans are replanned periodically (receding horizon), so only the first step is committed. Academic roots: build-order optimization in StarCraft by depth-first branch and bound over an abstracted economy (Churchill and Buro, AIIDE 2011), portfolio greedy search and playout-based evaluation (Churchill and Buro, AIIDE 2013), Monte Carlo tree search (Coulom 2006; Browne et al. survey 2012) and model-predictive control (Garcia, Prett and Morari 1989).

### 2a. Runtime rollout of an economy model

Each turn the AI builds several candidate action sequences from the enumerated config, rolls each forward H turns through an economy-only forward model (flows, queues, population cap, production slots, tech multipliers), scores the end state, and executes the first step of the best one. Opening quality and tech timing arise from search. The cost is a forward model that mirrors engine rules and a likely need for a Go primitive, since the AI DSL has no way to loop over plans.

### 2b. Generation-time offline search

The Python generator already reads the config, so it can run the forward model or `sim/` itself and emit the results (opening orders, tech timings, unit-mix tables, target curves) into the JSON as data. A changed config means a regenerate, with no engine change and no runtime step budget. The cost is that tables are computed for the configured scenario and are less adaptive to surprises, so they suit openings and priors better than reactive decisions.

## 3. Goal decomposition and planning

A top-level objective is recursively broken into subgoals using the preconditions and effects declared in the game's data, until every leaf is a concrete purchase or order with a deadline. Priority is slack (deadline minus delay minus now), so urgency rises as slack shrinks. Academic roots: STRIPS (Fikes and Nilsson 1971), hierarchical task network planning (Erol, Hendler and Nau 1994), goal-oriented action planning in games (Jeff Orkin, F.E.A.R., GDC 2006) and HTN in Killzone 2 (Humphreys, *Game AI Pro*, 2013).

### 3a. Backward chaining over the config graph

The root is required power by a time and the economy needed to pay for it. The generator walks `can_create`, `builds`, `population_support`, `affects_*` and tech prerequisites backward to produce a shortfall (target minus have minus in flight) with a deadline at every node. One convex urgency function of slack, defined once, yields demand per resource and per slot. This is close to v7's `goal_*` and `want_*`, computed generically from the config. The weak point is that the root targets are still estimated and techs are multipliers, which this structure expresses poorly.

### 3b. Role and archetype templates

Units and buildings are classified by derived roles (gatherer, builder, cheap mass, tank, ranged, siege, producer, supply, upgrade source) from their stats and `affects_*` fields, not ids. A small set of archetype plans (boom, timing attack, turtle, nomad relocation) gives parametric target curves per role, and a cost-effectiveness table picks the unit within a role. This is the hand-scripted tradition (Age of Empires II and StarCraft: Brood War scripted AIs, prioritized builder groups in Supreme Commander AIs) made config-aware. It is simple and tunable, and the least future-proof option because novel mechanics need new roles.

## 4. Linear and flow-network planning

The economy is modeled as a flow network of resources, workers, producers and sinks, and allocation is solved as an optimization problem rather than guessed. Academic roots: linear programming (Dantzig 1947), LP duality, where dual variables are exactly the shadow prices family 1 estimates, network flow (Ford and Fulkerson 1956), and the ratio planners used by factory games such as Factorio.

### 4a. Steady-state LP for rates

Solve for the worker split and production rates that maximize a power or growth objective subject to resource, population, producer-capacity and stamina constraints. The primal gives target allocations and the duals give resource prices to feed gathering and spending. This handles any config because the constraints are read from definitions. It models rates, not discrete timing, so it needs a second layer for what to buy this turn.

### 4b. LP as the price oracle for family 1

Use the LP only to compute shadow prices, then let a family 1 ledger make the discrete choices with those prices. This is the rigorous version of 1a's "scarcity" and replaces hand-set nonlinear urgency curves. The solver would run in the generator or offline, with its prices applied as expressions, or require a Go primitive for runtime solves.

## 5. Learned and optimized policies

A parametric policy is fit against the simulator by optimization or learning instead of being designed by reasoning. Academic roots: deep reinforcement learning at scale (AlphaStar, Vinyals et al., Nature 2019; OpenAI Five 2019), imitation learning, and black-box and evolutionary optimization (CMA-ES, Hansen and Ostermeier 2001).

### 5a. Parameter optimization of a designed policy

Keep the structure from families 1 to 4 and fit its `tune_*` parameters with CMA-ES or similar against the evaluator suite. This matches the existing generator workflow (`--list-parameters`, `name=value` overrides, `sim/` eval) and is the most realistic use of this family. The risk is overfitting to the configs, maps and opponents in the suite.

### 5b. Learned value function or policy

Train a model (reinforcement learning or imitation of the best scripted AI) to output purchase and allocation choices from the view. It is the most general in principle and the least interpretable, and it does not fit the JSON-rules runtime, so it would need a new model type and training infrastructure. Treat it as out of scope for v8.

## Scope: what these cover

Everything above is about the strategic and economic layer: what to buy, build and research, how many workers to put where, and when to save. It touches military only through demand: the value of power, the cost and delay of producing it, and a danger term that can pre-empt economic returns. It does not decide where armies go, when to fight, or unit-level actions.

Treat military, raiding, defense and micro as an orthogonal tactical layer with a narrow interface to the strategic one. The strategic layer exports demand (required power, production priorities, a threat estimate that can override economic value) and the tactical layer consumes the units it is given. Tactical layer families are different: Lanchester-style or ttk-based combat estimation, influence and potential maps for movement and threat, playout search (SparCraft, portfolio search) for engagements, and assignment problems for target selection and unit allocation. Those can be generalized from unit stats in the same config-driven way, and v7's `estimates.py`, `army.py` and `combat.py` are the starting point for that separate design.
