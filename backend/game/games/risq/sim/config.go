package main

import (
	"encoding/json"
	"os"
)

type ScenarioPlayer struct {
	AiConfig string `json:"ai_config"`
}

type Scenario struct {
	Seed             int64            `json:"seed"`
	Iterations       int              `json:"iterations"`
	BoardSize        int              `json:"board_size"`
	StartingDistance int              `json:"starting_distance"`
	MaxTurns         int              `json:"max_turns"`
	Players          []ScenarioPlayer `json:"players"`
}

func loadScenario(path string) (Scenario, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Scenario{}, err
	}
	var s Scenario
	if err := json.Unmarshal(data, &s); err != nil {
		return Scenario{}, err
	}
	if s.Iterations < 1 {
		s.Iterations = 1
	}
	if s.MaxTurns < 1 {
		s.MaxTurns = 300
	}
	return s, nil
}
