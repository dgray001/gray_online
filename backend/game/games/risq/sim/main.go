package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/dgray001/gray_online/game/games/risq"
	"github.com/dgray001/gray_online/util"
)

func main() {
	debug := flag.Bool("debug", false, "run a single replay (implies iterations=1)")
	quiet := flag.Bool("quiet", false, "discard engine stdout; keep only errors and the sim's own per-game log")
	seed_override := flag.Int64("seed", 0, "override the scenario's base seed")
	parallel := flag.Int("parallel", runtime.NumCPU(), "games to run concurrently (forced to 1 with -debug so its log stays in game order)")
	flag.Parse()
	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: sim [-seed N] [-debug] [-quiet] [-parallel N] <scenario_name>")
		os.Exit(1)
	}
	name := flag.Arg(0)
	if err := risq.LoadConfig("../config"); err != nil {
		log.Fatalf("loading risq config: %v", err)
	}

	scenario, err := loadScenario(filepath.Join("inputs", name+".json"))
	if err != nil {
		log.Fatalf("loading scenario: %v", err)
	}

	players := make([]PlayerConfig, len(scenario.Players))
	for i, p := range scenario.Players {
		players[i] = PlayerConfig{Nickname: fmt.Sprintf("ai-%d", i), ConfigPath: p.AiConfig}
	}

	iterations := scenario.Iterations
	if *debug {
		iterations = 1
	}
	base_seed := scenario.Seed
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

	workers := max(1, *parallel)
	if *debug {
		workers = 1
	}
	results := make([]Result, iterations)
	games := make(chan int)
	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			for i := range games {
				seed := base_seed
				if !scenario.FixedSeed {
					seed += int64(i)
				}
				start := time.Now()
				results[i] = RunGame(seed, players, scenario.Map, uint16(scenario.MaxTurns), 15*time.Minute)
				sim_log.Printf("game %d seed=%d turns=%d duration=%s error=%q",
					i+1, seed, results[i].Game.TurnNumber, time.Since(start), results[i].Error)
			}
		})
	}
	for i := range iterations {
		games <- i
	}
	close(games)
	wg.Wait()

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
