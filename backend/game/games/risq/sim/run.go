package main

import (
	"time"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/risq"
)

type PlayerConfig struct {
	Nickname   string
	ConfigPath string
}

type Result struct {
	Seed  int64
	Error string
	Game  risq.GameResult
}

func RunGame(seed int64, players []PlayerConfig, board_size int, starting_distance int, max_turns uint16, timeout time.Duration) Result {
	ai_players := make([]any, len(players))
	for i, p := range players {
		config := make(map[string]any, 2)
		config["nickname"] = p.Nickname
		config["config"] = p.ConfigPath
		ai_players[i] = config
	}
	settings := map[string]any{"ai_players": ai_players, "seed": float64(seed)}
	if board_size > 0 {
		settings["board_size"] = float64(board_size)
	}
	if starting_distance > 0 {
		settings["starting_distance"] = float64(starting_distance)
	}
	base := game.CreateBaseGame(uint64(seed), game.GameType_RISQ, settings)
	action_channel := make(chan game.PlayerAction, 16)
	r, err := risq.CreateGame(base, action_channel)
	if err != nil {
		return Result{Seed: seed, Error: err.Error()}
	}
	// mirrors LobbyRoom.gameBaseUpdates: nothing drains ViewerUpdates outside the real lobby
	done := make(chan struct{})
	defer close(done)
	go func() {
		for {
			select {
			case <-base.ViewerUpdates:
			case <-done:
				return
			}
		}
	}()
	game.Game_StartGame(r)
	deadline := time.After(timeout)
	for r.TurnNumber() < max_turns {
		select {
		case action := <-action_channel:
			r.PlayerAction(action)
		case <-base.GameEndedChannel:
			return Result{Seed: seed, Game: r.Results()}
		case <-deadline:
			return Result{Seed: seed, Error: "timed out", Game: r.Results()}
		}
	}
	return Result{Seed: seed, Game: r.Results()}
}
