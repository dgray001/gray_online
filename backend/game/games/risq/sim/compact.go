package main

import (
	"encoding/json"
	"log"
	"os"
	"strings"
)

// One game on one line, written by -quiet in place of the full results
type CompactGame struct {
	Sim      string
	Map      string
	Seed     int64
	Turns    uint16
	Error    string
	Ais      []string // the ai config in each seat
	Seats    []int    // the scenario player in each seat
	Outcomes []Outcome
	// each seat's outcome as every turn finished, one letter a turn (outcomeLetters is the key)
	Paths   []string
	Players []CompactPlayer
}

type CompactPlayer struct {
	Units         int
	Score         uint
	Land          int
	Eliminated    bool
	Kills         uint
	UnitsLost     uint
	Razes         uint
	BuildingsLost uint
}

func compactGame(scenario Scenario, sim SimResults, g SuiteGame) CompactGame {
	r := g.Result
	game := CompactGame{Sim: sim.Name, Map: sim.Map, Seed: r.Seed, Turns: r.Game.TurnNumber, Error: r.Error, Seats: g.Seats, Outcomes: r.Outcomes}
	for seat, p := range g.Seats {
		game.Ais = append(game.Ais, scenario.Players[p].AiConfig)
		var path strings.Builder
		for _, turn := range r.Trajectory {
			if seat < len(turn.Outcomes) {
				path.WriteString(outcomeLetters[turn.Outcomes[seat]])
			}
		}
		game.Paths = append(game.Paths, path.String())
		if seat < len(r.Game.Players) {
			pr := r.Game.Players[seat]
			game.Players = append(game.Players, CompactPlayer{pr.Units, pr.Score, pr.Land, pr.Eliminated, pr.Kills, pr.UnitsLost, pr.Razes, pr.BuildingsLost})
		}
	}
	return game
}

func writeCompact(path string, scenario Scenario, results Results) {
	f, err := os.Create(path)
	if err != nil {
		log.Fatalf("creating %s: %v", path, err)
	}
	defer f.Close()
	for _, sim := range results.Sims {
		for _, g := range sim.Games {
			line, err := json.Marshal(compactGame(scenario, sim, g))
			if err != nil {
				log.Fatalf("encoding a game of %s: %v", sim.Name, err)
			}
			f.Write(append(line, '\n'))
		}
	}
}
