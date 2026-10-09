package main

import (
	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/risq"
	"github.com/dgray001/gray_online/game/games/risq/internal/simoutcome"
	"time"
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
	// each seat's outcome from its own side, judged when the engine ends the game or the turn limit is reached
	Outcomes []Outcome
	// where the game stood as each turn finished, and each seat's outcome then
	Trajectory []TurnStanding
}

type TurnStanding struct {
	Turn     uint16
	Players  []risq.PlayerStanding
	Outcomes []Outcome
}

func RunGame(seed int64, players []PlayerConfig, map_name string, metrics []string, max_turns uint16, timeout time.Duration) Result {
	ai_players := make([]any, len(players))
	for i, p := range players {
		config := make(map[string]any, 2)
		config["nickname"] = p.Nickname
		config["config"] = p.ConfigPath
		ai_players[i] = config
	}
	extras := make([]any, len(metrics))
	for i, name := range metrics {
		extras[i] = name
	}
	settings := map[string]any{"ai_players": ai_players, "seed": float64(seed), "metrics": extras}
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
	var trajectory []TurnStanding
	last_turn := uint16(0)
	snap := func() {
		if t := r.TurnNumber(); t/5 > last_snap/5 {
			last_snap = t
			timeline = append(timeline, r.Snapshot())
		}
		if turn, standings := r.Standings(); standings != nil && turn > last_turn {
			last_turn = turn
			trajectory = append(trajectory, TurnStanding{Turn: turn, Players: standings, Outcomes: outcomes(standings)})
		}
	}
	for r.TurnNumber() < max_turns {
		select {
		case action := <-action_channel:
			r.PlayerAction(action)
			snap()
		case <-base.GameEndedChannel:
			results := r.Results()
			return Result{Seed: seed, Game: results, Timeline: timeline, Outcomes: finalOutcomes(results.Players), Trajectory: trajectory}
		case <-deadline:
			return Result{Seed: seed, Error: "timed out", Game: r.Results()}
		}
	}
	results := r.Results()
	return Result{Seed: seed, Game: results, Timeline: timeline, Outcomes: finalOutcomes(results.Players), Trajectory: trajectory}
}

type Outcome = simoutcome.Outcome

var (
	outcomeLetters = simoutcome.Letters
	outcomeOrder   = simoutcome.Order
	outcomes       = simoutcome.Outcomes
	finalOutcomes  = simoutcome.FinalOutcomes
)

const (
	OutcomeDefeat   = simoutcome.OutcomeDefeat
	OutcomeCrush    = simoutcome.OutcomeCrush
	OutcomeWinning  = simoutcome.OutcomeWinning
	OutcomeAhead    = simoutcome.OutcomeAhead
	OutcomeBehind   = simoutcome.OutcomeBehind
	OutcomeLosing   = simoutcome.OutcomeLosing
	OutcomeCrushed  = simoutcome.OutcomeCrushed
	OutcomeDefeated = simoutcome.OutcomeDefeated
)
