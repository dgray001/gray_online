# Risq map editor handoff

The core editor and Connections tool are implemented. Six feature gaps remain below. Automated checks passed; the latest changes have not been exercised in a browser.

## Goal

A dev-only, in-canvas map editor for Risq custom maps. Maps are saved as JSON files that the engine loads directly, so the sim and real games can run on them. It may become non-dev later, and may be reused for other games, so routes are namespaced per game.

## Map names

The `map` game setting is now explicit about its source:

- `custom:<name>` loads `backend/game/games/risq/config/maps/custom/<name>.json` (fixed map file).
- `script:<name>` loads `config/maps/scripted/<name>.json` (generator script).
- A bare name is an error. A missing file is a hard error (the old `default.json` fallback is gone).
- `create_game.go` defaults to `script:ring`. All files in `sim/inputs/*.json` (gitignored, local) were converted to `script:`.
- The two formats differ: a script is an array of `{step, params}`; a custom map is one object.

## Custom map format

Parsed strictly (`DisallowUnknownFields`). Backend types live in `internal/mapgen/custom.go` and `custom_starts.go`; frontend types live in `page_risq_map_editor/map_doc.ts`.

```json
{
  "board_size": 1,
  "players": 2,
  "player_start": {
    "size": 0,
    "terrain": 1,
    "spaces": [
      {
        "x": 0,
        "y": 0,
        "zones": [{ "x": 0, "y": 0, "units": [{ "id": 11, "count": 3 }] }]
      }
    ]
  },
  "spaces": [
    { "x": -1, "y": 0, "terrain": 1, "player_slot": 0 },
    { "x": 1, "y": 0, "terrain": 1, "player_slot": 1 }
  ],
  "regions": [
    {
      "name": "North",
      "gold_bonus": 3,
      "spaces": [
        [-1, 0],
        [1, 0]
      ]
    }
  ],
  "connections": [{ "from": [-1, 0], "to": [1, 0] }]
}
```

`players` is the map's maximum slot count (1 to 12). `player_slot` is a zero-based marker on a base-board space; every slot needs exactly one marker. With N players, only slots 0 through N-1 receive the shared `player_start`; unused slots retain their base contents. Template coordinates are relative to the home; its buildings and units receive the active player's identity automatically. `size` is the footprint radius, including 0 for a single space. Footprints must exist, be passable, and not overlap.

Legacy maps without a template still work: zone occupants specify `player`, and occupants above the active player count are skipped. Zones can contain units alongside a building or resource, but cannot contain both a building and resource. Only listed board spaces exist; omitted terrain defaults to 1. `starting_bank` is optional.

Connections are bidirectional. Omitting `direction` links center zones; values 0 to 5 join the source edge to the destination's opposite edge. Axial directions are `(1,0)`, `(1,-1)`, `(0,-1)`, `(-1,0)`, `(-1,1)`, `(0,1)`. Endpoints must be distinct and passable; an edge join cannot reuse an occupied edge.

Terrain IDs 151 (shallows), 152 (water), and 153 (deep water) are supported. Shallows are passable; water and deep water are impassable. Disconnected islands are valid, and connectivity is informational only.

Optional backend map rules are `starting_techs`, `unlimited_population`, `space_gold_income`, and `mercenaries_need_region`. These have no editor controls or explicit frontend type fields yet; loaded documents retain them at runtime. `source` provenance is not accepted by the backend yet.

## Backend

- `internal/mapgen/custom.go`: types, `parseCustomMap`, `loadCustomMap`, `GenerateCustomDocument` (in-memory), `buildCustomMap`, carve and fill helpers.
- `internal/mapgen/script.go`: `Generate` dispatches on the prefix; `generateScripted` runs a script.
- `create_game.go`: `CreateGame` wraps `createGame(g, ch, min_players, generate)` so the preview can build a game from a document with a minimum of 1 player.
- `internal/defs/config_load.go`: added `WriteConfigFile`, `ListConfigDir`, `RemoveConfigFile`.
- `dev_maps.go`: `PreviewCustomMap`, `ListCustomMaps`, `ReadCustomMap`, `SaveCustomMap` (builds the map first, only writes if the engine accepts it, re-indents), `DeleteCustomMap`, `AllTerrainConfigsToFrontend`. Map names must match `^[a-z0-9_]{1,40}$`.
- `PreviewCustomMap` builds a throwaway game with AI players, stops the AI, sets spy vision for pseudo-player -1 on every space, and returns `toFrontendFor(-1, 0, true)`. Without that vision step every space comes back unexplored.
- `backend/dev_routes.go` `registerRisqDevRoutes`, hooked from `main.go` only when `DEV` is true:
  - `GET /api/dev/risq/maps`, `GET` and `POST /api/dev/risq/maps/:name` (POST saves), `POST /api/dev/risq/maps/:name/delete`
  - `POST /api/dev/risq/preview` (body is a map document), `GET /api/dev/risq/terrains`
  - POST is used for save and delete because `scripts/api.ts` only has `apiGet` and `apiPost`.

## Frontend

URL: `https://localhost:8080/devmode/risq_map_editor/` (https only, trailing slash). Webpack entry `risq_map_editor` (`pages/risq_map_editor.ts`), static page `backend/static/devmode/risq_map_editor/index.html`, component `components/page_risq_map_editor/` (`dwg-page-risq-map-editor`).

Rendering reuse: the editor sends its document to the preview endpoint and feeds the returned snapshot to the game's own renderer. `rendering/draw_host.ts` defines `RisqDrawHost`, the narrow interface the draw code now takes instead of `DwgRisq` (`DwgRisq` still satisfies it; `tsc` was clean). `editor_host.ts` is the editor's implementation with stubbed selection, planning and armed state.

New generic canvas components, built like `DwgButton` / `DwgRectButton` (abstract state layer plus abstract rect layer, concrete subclasses override hooks):

- `canvas_components/input/input.ts` (`DwgInput`) and `rect_input.ts` (`DwgRectInput`): focus, caret, typing, backspace and delete, arrows, home and end, enter, escape, paste. `max_length`, `allowed_chars`. Key events use a capture-phase document listener only while focused.
- `canvas_components/dropdown/dropdown.ts` (`DwgDropdown`) and `rect_dropdown.ts` (`DwgRectDropdown`): single select, string values, popover queued through `queueTooltipDraw`, wheel scroll with a non-draggable thumb.
- `dropdown/multi_select_dropdown.ts` in the same folder is the user's own untracked work. Do not touch it.

Editor files in `page_risq_map_editor/`:

- `page_risq_map_editor.ts`: composition root. Owns the document, history, tools, panels, preview scheduling, selection, hotkeys.
- `map_editor_header.ts`: in-canvas header (map dropdown, name input, size input, New, Save, Delete, status text). Save is shown for an open document; Delete is shown for a saved map. The backend validates documents before saving.
- `editor_right_panel.ts`: collapsible right panel; top half tool buttons, bottom half the active tool's config.
- `editor_left_panel.ts`: selection info; shown only when Select picks something, closed by the x button or by changing tool.
- `editor_tool.ts`: abstract `EditorTool`, `EditorToolContext`, `EditorConfigPanel`, `EditorTarget`.
- `edit_history.ts`: undo and redo of `EditCommand`s (tools apply to the document themselves, then `record` the command).
- Tools: Select, Terrain, Spaces, Zones, Slots, Regions, Links, Validate. Each lives under `tools/` with its configuration panel or palette.
- `tools/connections_tool.ts` and `connections_panel.ts`: Links (C). Click two endpoints to add; repeat the same ordered pair and mode to remove. Reversed edge endpoints require the opposite direction. Clicking the same space twice cancels. The mode button cycles center links and edge directions 0 to 5. Existing links and pending endpoints are drawn; edge markers show direction. Undo/redo, reversed endpoints, water rejection, and occupied-edge checks are implemented.
- `map_doc.ts`, `map_api.ts`, `editor_host.ts`, `editor_style.ts`, `editor_button.ts`.

Behavior decisions already made with the user:

- New takes only a name and a size (a new map starts with 1 slot). Slot count and starting bank are edited with the Slots tool in the right panel, not the header. Start-marker placement is still missing.
- Selecting a map in the dropdown loads it. Nothing loads by default.
- Delete asks for a second press. Names must match the pattern above and not already exist when creating.
- Terrain painting is optimistic: the local snapshot space is patched immediately, the server preview confirms. Edit previews are coalesced; revision guards discard stale preview and map-load responses, including stale errors.
- Paint tools consume left-drag. Pan with middle-drag or arrow keys. The Select tool leaves left-drag as pan and selects on release if the mouse did not drag.
- Hotkeys: V Select, B Terrain, S Spaces, Z Zones, L Slots, R Regions, C Links, X Validate; Esc returns to Select. Ctrl+Z undoes; Ctrl+Y or Ctrl+Shift+Z redoes. Hotkeys are off while typing in the header.

## Browser verification still required

1. Run the current Go backend and frontend dev build, then open the URL above. Restart an older backend if the dev routes are missing.
2. Check header typing/dropdowns, panel layout/collapse, panning, terrain painting/brush sizes, palette scrolling, selection, New, Save, Delete, and undo/redo across tools.
3. Check Links: add/remove center and edge joins, cancel a pending endpoint, change modes, switch maps during pending previews, and verify labels/highlights stay readable at different zoom levels.
4. Open a shared-start map, reduce slots and undo, and save/reload without changing its template or rule fields. Islands should produce no connectivity error.

## Remaining features

- Shared-start editing: add controls for the common `player_start` footprint, terrain, units, buildings, and resources. Current tools edit the base board; template application replaces base contents inside active footprints in the preview. The editor needs a clear distinction between base-board and template editing.
- Slot placement/reordering: add board controls to place, move, and order `player_slot` markers, with footprint and overlap feedback. Reducing the slot count already removes excess markers without erasing their base spaces. Increasing it does not create markers and leaves a shared-start map invalid until markers are supplied.
- Symmetry mode that rotates edits to every start slot (use the generator's `rotateAxial` semantics).
- Script import/provenance: implement a generation endpoint taking `{script, seed, players}`, serialize the generated board into a custom document, and add backend support for a `source` field before emitting it. The proposed route is `POST /api/dev/risq/maps/generate`; it does not exist yet.
- Playtest button (`custom:<name>` already works in the sim and in games).
- Unsaved-changes tracking: indicate when edits have not been saved and handle switching maps or leaving with pending edits. There is no dirty-state indicator or navigation guard yet.

## Verification completed

- Full frontend TypeScript check passed (`cd frontend && ./node_modules/.bin/tsc --noEmit`). Scoped ESLint and Prettier checks passed for the editor. Use checks without `--fix` or `--write` when reviewing only.
- In-memory behavior checks passed for all ten geographic maps, shared-start/military-only validation, valid islands, water/shallows and explicit links, slot reduction, duplicate region names, conflicting zone occupants, erase cleanup/protected starts, undo/redo, and stale preview/load responses.
- Connections behavior checks passed for center add/remove, all mode cycling, direction 0, reversed seam removal, invalid endpoints, occupied natural/seam edges, cancellation, document switching, panel layout, and overlay positions/state isolation.
- Backend `go test ./...` passed during the review. There are no permanent Risq tests, so this does not replace behavior checks or browser verification.

## Known limits

- A click in an input puts the caret at the end; there is no text selection.
- The dropdown scroll thumb cannot be dragged.
- Every edit round-trips to the server for rendering (only terrain is patched locally).
- Preview instantiates all map slots; there is no preview control for a smaller active-player count. Shared-start and inactive-slot behavior is supported by the backend, but the editor cannot display that smaller-player scenario yet.
- Local validation checks starts, connectivity, and hardcoded unit/building IDs; it is not a complete schema or fairness validator. Backend preview/save remain authoritative for clipped/overlapping footprints, connection validity, regions, and configuration IDs.
- No permanent backend tests cover `PreviewCustomMap`, save or delete. A throwaway test confirmed the preview returns buildings, units and resources and that bad documents and bad names return clean errors.

## Environment notes

- The webpack dev server serves HTTPS on 8080 and proxies `/api` to the Go server on 6807. In `DEV` the backend sets `PORT=6807` itself, so a second backend cannot be started alongside the user's.
- Headless checking worked with Firefox 156 over WebDriver BiDi driven by a small Node script (`--remote-debugging-port`, `browsingContext.captureScreenshot`, `script.evaluate` to click elements). The script lived in the session scratchpad and is not in the repo.
- Sim check: `cd backend/game/games/risq/sim && go run . -quiet <scenario>` with `"map": "custom:tiny"` or `"script:ring"` in `inputs/<scenario>.json` both ran clean to turn 40.

## User preferences that matter here

These come from `~/.claude/CLAUDE.md` and from this session:

- No comments in code (the user objected to comments being added everywhere). Self-documenting names and types instead.
- Follow existing patterns. New generic UI pieces are abstract classes with abstract hooks, like `DwgButton`.
- Mandatory types on parameters and returns. No em dashes, no emojis, no flattery, terse updates.
- Keep edits small and explain them before making them. Do not edit files from scripts unless told to.
- Do not run git commands unless asked. Commit message is the README "Current Version" line, updated first (`vX.Y<letter>: <5 words max>`).
- Do exactly what was asked. The user interrupted several times for extra UI or assumptions they had not requested (slot controls in the header, extra inputs on New).
- If a design question is open, ask or give a recommendation, do not silently pick.

## Stale docs

`.claude/risq.md` and `.claude/risq_ai.md` still describe the removed `mirror` script step. Use current source for map scripting behavior.
