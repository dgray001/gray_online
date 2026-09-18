# Map scripting (design)

Not yet implemented. This captures the agreed design for risq's terrain/board generation system before any code is written, so implementation can proceed without re-deriving the decisions below.

## Goal

Replace `CreateGame`'s current hardcoded generation (every space is `defaultTerrainId`, resources placed via two fixed hardcoded passes) with a data-driven system modeled on Age of Empires' Random Map Scripts: a small set of composable primitive steps, and a "map script" is just an ordered list of calls to them with arbitrary count/order/params. New map types (analogous to AoE's Arabia/Black Forest/Islands) become new JSON files, never new Go code.

## Execution model

A map script is a JSON array of `{step, params}` entries, interpreted in order by a runner that dispatches each entry by `step` name to a registered Go function. Each step function has its own typed param struct, decoded from `params` the same way `TerrainConfig`/`BuildingConfig`/etc. are already decoded — panics on malformed config, consistent with the rest of risq's config loading.

Every step function receives a shared execution context (board, seeded `*rand.Rand`, `num_players`, `board_size`/`starting_distance`) so scripts stay fully deterministic given a seed. `board_size`/`starting_distance` remain engine-computed from player count (a fairness/balance decision), not script-controlled, and are just handed to the script as more ambient inputs alongside `num_players`.

The context also holds mutable state that some steps *produce* and later steps *consume*:
- player-start positions/orientations (written once by `player_starts`)
- named region membership (written incrementally by any shaping step that opts in)

This is why steps aren't fully order-independent: anything referencing player starts or a region name must run after the step(s) that populate them.

## File layout

- `risq_map_script.go` — script JSON shape, the `config/maps/` `embed.FS` + name-based lookup (mirrors `risq_player.go`'s AI-config pattern: `embed.FS` over `config/maps/*`, selected by name, falling back to `config/maps/default.json`), and the orchestration function `CreateGame` calls to run a parsed script.
- `risq_map_steps.go` — step implementations + the name→function registry. Grows as steps are added; the other file shouldn't need to change when it does.

## Primitives

**Shaping:**
- `terrain_fill(terrain_type)` — whole-board base type. Runs first in most scripts.
- `terrain_blob(terrain_type, seed_count, size, min_spacing)` — N blob-grown patches of one type. Covers hills, mountains, swamp pockets, water regions — same function for all of them, just a different `terrain_type` arg. There is no separate "elevation" concept the way AoE has it; `hilly` etc. are ordinary `terrain_type`s in our model, not an overlay on top of a base type.
- `terrain_line(terrain_type, width, ...)` — connected strip between two points (rivers, mountain ranges), distinct from a blob because it's path-shaped rather than round.
- `terrain_border(terrain_type, width)` — a ring around the map edge.

**Fixups:**
- `ensure_connectivity()` — verifies (and patches, if needed) that random shaping hasn't split the board into unreachable islands. Necessary now that water is impassable; an unlucky blob roll could otherwise wall off a player.

**Resources:**
- `resource_scatter(chance, category_weights)` — background per-zone probability roll (generalizes today's hardcoded `placeUniformResources`).
- `resource_cluster(resource_id, seed_count, size)` — blob-style placement for dense resource patches, distinct from scatter.
- `resource_min_spacing(distance)` — spacing constraint across placements, not a placement itself.

**Symmetry:**
- `mirror(step)` — wraps another step and replays its random choices rotated to every player-start direction, so a feature placed "near a player" appears fairly near all of them instead of being independently re-rolled per player.

**Regions (future-proofing, not implemented yet):** rather than a dedicated region step, `region` is an optional param any shaping primitive can carry alongside `terrain_type`. The interpreter accumulates "these spaces belong to region X" into the context's region map as steps run. A region can span multiple steps/shapes (a hilly blob here, a swamp line there, same `region` name) — matches how Risk continents aren't uniform terrain. No new step type or plumbing beyond one more context field and one more optional param on existing primitives.

## `player_starts`

`player_starts(pattern, area_size, terrain_type, resources, buildings)`

Not just a position generator — a self-contained "stamp the home area" primitive:
- `pattern` — placement pattern for the `num_players` symmetric positions (`"ring"` matches today's spaced-directions-around-center behavior; room for `"scattered"`/`"opposite_pairs"` later).
- `area_size` — footprint radius around each start.
- `terrain_type` — forced terrain for the whole footprint, overwriting whatever earlier shaping steps put there. This auto-clear means `player_starts` can run at any point in the script relative to terrain shaping and still wins within its own footprint.
- `resources` / `buildings` — the guaranteed resource set and starting building(s) for the area, replacing the hardcoded 5-node set (`placeFixedResourceSet`) and hardcoded `building_id: 1` (`createPlayerStart`) that exist today.

There is no separate `clear_region` step; this subsumes it.

**Overlap rule, easy to get wrong:** `area_size` and the pattern's start-to-start distance are independent. If `area_size` is large relative to spacing, neighboring players' *areas* can and will overlap — that's allowed. The only hard non-overlap constraint is the literal starting space (the exact zone holding the home building/units) — those must never coincide between players. When areas overlap, each player's resources/buildings/terrain-forcing must still be applied in full and independently; implementation must loop per-player over that player's own footprint, not process a merged/deduped set of "already-handled" spaces, or a second player's placements in the overlap would silently get skipped.

Still writes into the context's start-position/orientation state for `mirror` and anything else placing features relative to player starts, even though nothing needs to separately "clear around" them anymore.
