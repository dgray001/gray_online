# Local Fiddlesticks goldset

From `model_training/`, run `go run ./goldset/cmd` and open http://localhost:7077.
The server binds to localhost and creates 20 pending situations on first start.
Use `-port`, `-data`, `-web`, or `-images` to override local paths and port.

With the server stopped, `go run ./goldset/cmd -add-foundation` adds the curated
30-item foundation batch once: 15 bidding and 15 card-play situations based on
the first annotations. Eight have rule-provable answers. Their private answer
key and case manifest live in `goldset/data/batches/`; neither is served by the
annotation API or included in training exports. Existing labels stay unchanged.

Choose a bid or legal card, optionally enter confidence and reasoning, then save.
The Labelled view allows revisions. Skip leaves an item open. Add situations
generates reproducible cases filtered by player count, hand size, phase, and scoring.
Only your explicit choices become training targets. Exports also retain the frozen
Model4 reference separately, so reasoning about its suggestions remains interpretable.

Data lives in `goldset/data/items.json` and `goldset/data/labels.json`.
Writes are atomic; label revisions are retained. Export uses the latest revision.
The export includes schema version 2, 740 features, legal action IDs, state,
source seeds, confidence, notes, and your action. The objective is winning the game.

Every label revision includes the displayed state's snapshot, Model4 sampled
action, explanation values, capture time, and SHA-256 fingerprint. New items
also capture Model4's actual parameter values. Existing items are never
recomputed when Model4 changes. Old labels are backfilled from their original
stored items, without changing notes/actions/confidence/timestamps; the original
labels file is kept as `labels.before-reference.json`. Unknown legacy model
parameters remain null. Reference fields are annotation context, not targets.

Label schema 2 adds `acceptable_actions`, the set of equally acceptable answers.
The original click remains in `action` / `target_action` for compatibility.
Training and evaluation must credit any member of the acceptable set, rather
than penalizing alternatives or requiring a uniform probability distribution.
Confidence remains separate from equivalence.

Enable Multiple equally acceptable answers in the UI to select a set.
Single-answer labels export a singleton set. A one-time `equivalences.json`
correction can upgrade existing labels; originals are retained in
`labels.before-equivalences.json`, and future edits are not overwritten.

After frontend edits, run `npm run build:goldset` from `frontend/`.
The UI uses real card assets. Model4 explanations expose its own computations;
one-card odds are exact under the stated uniform-unknown-card assumption.
The RL reward remains unchanged; reward redesign is deferred.
