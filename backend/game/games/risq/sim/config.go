package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
)

type ScenarioPlayer struct {
	AiConfig string `json:"ai_config"`
}

type Scenario struct {
	Seed       int64            `json:"seed"`
	FixedSeed  bool             `json:"fixed_seed"`
	Iterations int              `json:"iterations"`
	Map        string           `json:"map"`
	MaxTurns   int              `json:"max_turns"`
	Players    []ScenarioPlayer `json:"players"`
	// opt-in extra metrics beyond the core set, e.g. "production"
	Metrics []string `json:"metrics,omitempty"`
	// A suite: every bucket runs games_per_bucket games of these players on its own map, and the run reports
	// how each player's AI did overall and per bucket
	Buckets        []SuiteBucket `json:"buckets,omitempty"`
	GamesPerBucket int           `json:"games_per_bucket,omitempty"`
	// alternate seats game by game so neither AI always moves first or starts in the same spot
	SwapSeats bool `json:"swap_seats,omitempty"`
}

type SuiteBucket struct {
	Name     string         `json:"name"`
	Map      string         `json:"map"`
	Settings map[string]any `json:"settings,omitempty"`
	// overrides the suite's max_turns for this bucket when set
	MaxTurns int `json:"max_turns,omitempty"`
	// plays exactly this seed (with swap choosing the seat order) instead of the seed and seats derived from the bucket and game index
	Seed int64 `json:"seed,omitempty"`
	Swap bool  `json:"swap,omitempty"`
	// total seats, shared evenly by the suite's players, whose best performers are compared; defaults to one seat each
	Players int `json:"players,omitempty"`
	// overrides the suite's games_per_bucket for this bucket when set
	Games int `json:"games,omitempty"`
}

func (b SuiteBucket) seatCount(sides int) int {
	if b.Players > 0 {
		return b.Players
	}
	return sides
}

func (b SuiteBucket) gameCount(default_games int) int {
	if b.Games > 0 {
		return b.Games
	}
	return default_games
}

func loadScenario(path string) (Scenario, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Scenario{}, err
	}
	var s Scenario
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&s); err != nil {
		return Scenario{}, err
	}
	if s.Iterations < 1 {
		s.Iterations = 1
	}
	if s.MaxTurns > math.MaxUint16 {
		return Scenario{}, fmt.Errorf("max_turns %d exceeds %d", s.MaxTurns, math.MaxUint16)
	}
	if s.MaxTurns < 1 {
		s.MaxTurns = 300
	}
	for _, b := range s.Buckets {
		if b.Players > 0 && len(s.Players) > 0 && b.Players%len(s.Players) != 0 {
			return Scenario{}, fmt.Errorf("bucket %q: players %d is not a multiple of the %d suite players", b.Name, b.Players, len(s.Players))
		}
	}
	return s, nil
}
