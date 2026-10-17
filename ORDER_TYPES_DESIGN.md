# Risq order types

Orders have exactly two shapes on the frontend and exactly two on the backend. They never mix: no optional `internal_id`, no defaulting a missing id to 0, no list that holds both.

## The two types

| | Unreceived | Received |
|---|---|---|
| Meaning | Player-authored, not yet picked up by the server's resolver | Server-owned, has an id, may span turns |
| Frontend | `RisqFrontendOrder` | `RisqOrder` |
| Backend | `defs.OrderFromFrontend`, stored as-is in `RisqPlayer.unreceived_orders` | `RisqOrder` in `RisqPlayer.active_orders` (`toFrontend()`) |
| `internal_id` | Not a field | Required |

Shared fields: `player_id`, `order_type`, `target_id`, `subjects`, `clear_previous_orders`.

`turn_received` and `turn_resolved` stay on the backend `RisqOrder` (state hash) but are not sent: no frontend type declares them and nothing on the frontend reads them. `RisqOrder.received` is deleted, since list membership now encodes it.

## Current problems

1. The submit echo and the snapshot send one `active_orders` list holding both kinds. Unreceived orders are serialized as `RisqOrder` with `internal_id` omitted when it is 0.
2. `RisqFrontendOrder` declares `internal_id?`, which blends the two types.
3. `serverToRisqOrder` does `internal_id ?? 0`, forcing unreceived orders into the received type.
4. `revertSubmittedToPending` tests `internal_id === undefined`, which never matches those 0s, so unsubmit drops the orders.
5. `RisqPlayer.active_orders` duplicates `orders_model.submitted`. It is only read to seed `setSubmitted` and by one lookup in `order_row_resolve.ts`.
6. Backend `RisqPlayer.active_orders` mixes both kinds, split by `RisqOrder.received`. Echoing unreceived orders from the subject map also scrambles subject order (Go map iteration).
7. While submitted, `orders_model` clears `pending`, and cancellation, spending, building-limit and foundation logic read only `pending`. Spending drops to 0 and strikethroughs vanish until the turn resolves (PLANS.md: "Resources spent this turn don't show while orders submitted and waiting").

## Wire format

Everywhere the server sends a player's orders (the `submitted-orders` / `unsubmitted-orders` notice and the snapshot's player data), `active_orders` is replaced by two fields:

- `received_orders`: `RisqOrder` shape, `internal_id` always present, no turn fields.
- `unreceived_orders`: `OrderFromFrontend` shape, no id and no turn fields, subjects in submitted order.

Inbound `submit-orders` is unchanged: `{orders: OrderFromFrontend[]}`.

## Backend changes

- `player.go`: add `unreceived_orders []defs.OrderFromFrontend`. `active_orders` now holds only received orders. Serialize the two lists separately (`received_orders` via `toFrontend()`, `unreceived_orders` as stored) and update the snapshot call site (`player.go:363`). `receivePlayerOrder` (`player.go:242`) keeps searching `active_orders` for cancel targets.
- `order.go`: delete `RisqOrder.received` and the `internal_id != 0` branch and turn fields in `toFrontend`.
- `actions.go`: `executeSubmitOrders` stores the orders in `unreceived_orders` with no subject lookup. `executeUnsubmitOrders` sets it to nil. `broadcastSubmission` sends both fields to the submitter; other players still get no orders.
- `tick.go`: `resolveActiveOrders` (`:29-35`) builds each `RisqOrder` from `unreceived_orders`, resolving subjects there, assigning `nextOrderInternalId()`, and appending to `active_orders` through `deliverOrder`. Resolve subjects with the authenticated `player.Player_id`, not the client-supplied `o.Player_id`. Update the `order.received` checks at `:30` and `:133`, and clear `unreceived_orders` after delivery (`order.go:50`).
- `state_hash.go`: drop `received` from `hashOrders` (`:52`) and hash `unreceived_orders` (`:126`).
- `internal/aibridge/snapshot.go:60`: the player struct reads `received_orders` instead of `active_orders`. Otherwise `v.order_ids` (`view.go:97`) stays empty and `convert.go` drops every unit and building order from the AI view. The per-unit and per-building `active_orders` at `:78` and `:101` are unchanged.
- Tests: update fixtures and assertions that read `active_orders` (`updates_test.go` and others found by grep) and the `orderShape` string at `fog_test.go:159`.

## Frontend changes

- `transport/snapshot_types.ts`: `RisqPlayerFromServer` and `SubmissionNoticeData` carry `received_orders: RisqOrderFromServer[]` (required `internal_id`) and `unreceived_orders: RisqFrontendOrder[]`.
- `model/types.ts`:
  - `RisqFrontendOrder` has no `internal_id`.
  - `RisqOrder` keeps a required `internal_id`.
  - `RisqPlayer.active_orders` is removed; the model's `received` list replaces it.
- `transport/snapshot_conversion.ts`: `RisqOrderFromServer` differs from `RisqOrder` only by `order_type` typing and an optional flag, so `serverToRisqOrder` shrinks to a typed cast with no `?? 0`. Unreceived orders need no conversion.
- `application/orders/orders_model.ts`: two lists, `received: RisqOrder[]` and `unreceived: RisqFrontendOrder[]`.
  - `unreceived` is both the editable pending list (unsubmitted) and the submitted list (submitted); `canGiveOrders()` already blocks edits while submitted. Submit no longer calls `clearPending()`, so cancellation, spending, building-limit and foundation logic (`isExplicitlyCancelling`, `isCancelling`, `updateResourceSpending`, `atBuildingLimit`, `hasPlannedFoundation`, `syncLocalFoundations`) keep working while waiting for other players. This also fixes problem 7.
  - `revertSubmittedToPending` and `pending`/`submitted` as separate lists are removed.
  - `setReceived(orders)` replaces `received`. `setUnreceived(orders)` is called only when `orders_submitted` is true (reconnect or submit echo). Snapshots received while unsubmitted must not touch `unreceived`, so unrelated snapshots keep local pending orders.
  - `all()` and the effective-order logic work on `RisqOrder | RisqFrontendOrder`, discriminated by `'internal_id' in order` (replacing every `internal_id === undefined` test in the model).
- `application/updates.ts`: `applySubmission` and `setNewGameData` pass the two wire lists to the model. In `applySubmission`, call `setReceived`/`setUnreceived` before `submission.submitted()`/`unsubmitted()` so the single refresh sees current data. `unsubmitted()` no longer reverts anything; the notice's `unreceived_orders: []` is not applied while unsubmitted.
- `application/session.ts`: `applySubmission` patches both lists on the raw snapshot.
- `application/replay/replay_snapshot.ts`: the three `active_orders: []` literals (`:41`, `:59`, `:154`) become `received_orders: []` and `unreceived_orders: []`.
- Narrowing to `RisqOrder | RisqFrontendOrder` with `'internal_id' in order`: `resolveOrderRow`, `drawOrdersForUnit`, `lastAreaAttack` (its `Pick<RisqFrontendOrder, 'internal_id' | ...>` parameter), and `RisqOrderFilter.matches`.
- `canvas_components/order_row/order_row_resolve.ts:286`: look up the cancel target in the model's `received` list, not in `player.active_orders`.

## Decisions

- Frontend `RisqPlayer.active_orders` is deleted. Planning, overlays, and the ledger already read the model and the per-unit and per-building `active_orders`.
- The model holds two lists, not three, so submitted orders stay visible to spending and cancellation logic.

## Order of work

1. Backend split (including AI snapshot and state hash) and tests.
2. Frontend types and conversion (including replay snapshot).
3. `orders_model` and its callers, including the narrowing sites.
4. Remove frontend `RisqPlayer.active_orders`.
5. Run `gofmt -w .` and `go test ./...` from `backend/`, and `npm run check` from `frontend/`.
