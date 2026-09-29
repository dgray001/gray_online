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
	return s, nil
}
