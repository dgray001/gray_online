package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dgray001/gray_online/game/games/risq"
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

func resolveAiConfig(inputs_dir, name string) (map[string]interface{}, error) {
	data := risq.DefaultAiConfigJSON
	if name != "" && name != "default" {
		read, err := os.ReadFile(filepath.Join(inputs_dir, name))
		if err != nil {
			return nil, err
		}
		data = read
	}
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing ai config %q: %w", name, err)
	}
	return config, nil
}
