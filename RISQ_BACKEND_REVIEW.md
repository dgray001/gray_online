# RISQ backend review

Open findings from the read-only review of `backend/game/games/risq`. Every other finding from the original review has been fixed or dropped.

## Findings

### 1. Final game state is not broadcast to players when elimination ends the game

`risq.go:100-108` updates elimination and reports, then `checkWinCondition` ends the game. The winning resolution never calls `startNextTurn`, so no final `start-turn` snapshot is sent. In an integration run, the eliminated player's last queued update still reported `eliminated=false`, although `Results()` immediately afterward reported true and the base game was ended.

Send a final state/report update before ending the game, or make the game-over message carry a final authoritative snapshot.

Verified: `EndGame` sets `game_ended` first, and `AddUpdate` refuses to send after that, so the final update has to go out before `EndGame`.

### 2. AI configuration errors can silently become a no-op model

AI config parse failures can become `NoopModel` (`player.go:createAiModel`), hiding configuration errors and producing empty turns. Return structured errors with config/action indexes.

Verified, with "silently" too strong:
- Unreadable and malformed configs print to stderr.
- The JSON unmarshal failure path has `// TODO: log error` and calls `ParseModel(nil)`, which hits the missing-`rules` message, so the real parse error is lost.
- There is no structured error, and game creation does not fail.
