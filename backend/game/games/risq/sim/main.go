package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/dgray001/gray_online/util"
)

func main() {
	debug := flag.Bool("debug", false, "run a single replay (implies iterations=1)")
	quiet := flag.Bool("quiet", false, "discard engine stdout; keep only errors and the sim's own per-game log")
	seed_override := flag.Int64("seed", 0, "override the scenario's base seed")
	flag.Parse()
	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: sim <scenario_name> [-seed N] [-debug] [-quiet]")
		os.Exit(1)
	}
	name := flag.Arg(0)

	scenario, err := loadScenario(filepath.Join("inputs", name+".json"))
	if err != nil {
		log.Fatalf("loading scenario: %v", err)
	}

	players := make([]PlayerConfig, len(scenario.Players))
	for i, p := range scenario.Players {
		players[i] = PlayerConfig{Nickname: fmt.Sprintf("ai-%d", i), ConfigPath: p.AiConfig}
	}

	iterations := scenario.Iterations
	base_seed := scenario.Seed
	if *debug {
		iterations = 1
	}
	if *seed_override != 0 {
		base_seed = *seed_override
	}

	out_dir := filepath.Join("outputs", name)
	if err := os.MkdirAll(out_dir, 0755); err != nil {
		log.Fatalf("creating output dir: %v", err)
	}

	// sim.log: the sim CLI's own lines (always). In -quiet mode it also absorbs engine errors.
	sim_log_file, err := os.Create(filepath.Join(out_dir, "sim.log"))
	if err != nil {
		log.Fatalf("creating sim.log: %v", err)
	}
	defer sim_log_file.Close()
	sim_log := log.New(sim_log_file, "", log.LstdFlags)

	// fmt.Println(...) everywhere in the process reads os.Stdout per call
	var game_log_file *os.File
	if *quiet {
		devnull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
		if err != nil {
			log.Fatalf("opening devnull: %v", err)
		}
		defer devnull.Close()
		os.Stdout = devnull
	} else {
		// game.log: the engine's own normal output, only when not -quiet.
		game_log_file, err = os.Create(filepath.Join(out_dir, "game.log"))
		if err != nil {
			log.Fatalf("creating game.log: %v", err)
		}
		defer game_log_file.Close()
		os.Stdout = game_log_file
	}

	// errors always go to sim.log, plus game.log too when it exists
	stderr_cleanup, err := util.RedirectStderr(sim_log_file, game_log_file)
	if err != nil {
		log.Fatalf("redirecting stderr: %v", err)
	}
	defer stderr_cleanup()

	if *debug {
		// debug.log: ai decision + combat tick logs, separate from game.log/sim.log.
		debug_log_file, err := os.Create(filepath.Join(out_dir, "debug.log"))
		if err != nil {
			log.Fatalf("creating debug.log: %v", err)
		}
		defer debug_log_file.Close()
		util.DebugLog.SetOutput(debug_log_file)
	}

	results := make([]Result, 0, iterations)
	for i := 0; i < iterations; i++ {
		seed := base_seed
		if !scenario.FixedSeed {
			seed += int64(i)
		}
		start := time.Now()
		result := RunGame(seed, players, scenario.Map, uint16(scenario.MaxTurns), 30*time.Second)
		sim_log.Printf("game %d seed=%d turns=%d duration=%s error=%q",
			i+1, seed, result.Game.TurnNumber, time.Since(start), result.Error)
		results = append(results, result)
	}

	results_file, err := os.Create(filepath.Join(out_dir, "results.json"))
	if err != nil {
		log.Fatalf("creating results.json: %v", err)
	}
	defer results_file.Close()
	encoder := json.NewEncoder(results_file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(results); err != nil {
		log.Fatalf("writing results.json: %v", err)
	}

	sim_log.Printf("ran %d game(s), results in %s", iterations, out_dir)
}
