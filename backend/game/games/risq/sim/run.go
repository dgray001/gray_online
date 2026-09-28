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
	Seed     int64
	Error    string
	Game     risq.GameResult
	Timeline [][]risq.PlayerSnapshot
	// index of the player whose opponent's army was wiped out (-1 if none); the game is stopped there
	Crush int
}

func RunGame(seed int64, players []PlayerConfig, map_name string, max_turns uint16, timeout time.Duration) Result {
	ai_players := make([]any, len(players))
	for i, p := range players {
		config := make(map[string]any, 2)
		config["nickname"] = p.Nickname
		config["config"] = p.ConfigPath
		ai_players[i] = config
	}
	settings := map[string]any{"ai_players": ai_players, "seed": float64(seed)}
	if map_name != "" {
		settings["map"] = map_name
	}
	base := game.CreateBaseGame(uint64(seed), game.GameType_RISQ, settings)
	action_channel := make(chan game.PlayerAction, 16)
	r, err := risq.CreateGame(base, action_channel)
	if err != nil {
		return Result{Seed: seed, Error: err.Error()}
	}
	defer r.StopAi()
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
	var timeline [][]risq.PlayerSnapshot
	last_snap := uint16(0)
	crush := -1
	snap := func() {
		if t := r.TurnNumber(); t/5 > last_snap/5 {
			last_snap = t
			cur := r.Snapshot()
			timeline = append(timeline, cur)
			if t > 40 && len(cur) == 2 {
				inf := func(p risq.PlayerSnapshot) int { return p.Units[11] + p.Units[12] + p.Units[13] }
				for i := 0; i < 2; i++ {
					if inf(cur[1-i]) == 0 && inf(cur[i]) >= 20 && crush < 0 {
						crush = i
					}
				}
			}
		}
	}
	for r.TurnNumber() < max_turns {
		select {
		case action := <-action_channel:
			r.PlayerAction(action)
			snap()
			if crush >= 0 {
				return Result{Seed: seed, Game: r.Results(), Timeline: timeline, Crush: crush}
			}
		case <-base.GameEndedChannel:
			return Result{Seed: seed, Game: r.Results(), Timeline: timeline, Crush: -1}
		case <-deadline:
			return Result{Seed: seed, Error: "timed out", Game: r.Results()}
		}
	}
	return Result{Seed: seed, Game: r.Results(), Timeline: timeline, Crush: -1}
}
