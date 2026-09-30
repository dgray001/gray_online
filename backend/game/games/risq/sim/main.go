package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

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
	// the summary goes to the terminal even when engine output is redirected
	terminal := os.Stdout
	if err := risq.LoadConfig("../config"); err != nil {
		log.Fatalf("loading risq config: %v", err)
	}

	scenario, err := loadScenario(filepath.Join("inputs", name+".json"))
	if err != nil {
		log.Fatalf("loading scenario: %v", err)
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
	// a plain scenario is one sim on its map; a suite lists several, each on its own map
	sims := scenario.Buckets
	games_per_sim := scenario.GamesPerBucket
	if len(sims) == 0 {
		sims = []SuiteBucket{{Name: name, Map: scenario.Map}}
		games_per_sim = iterations
	} else if *debug {
		log.Fatalf("-debug replays one game; run one of the suite's maps as a plain scenario instead")
	}
	runSims(scenario, sims, max(1, games_per_sim), out_dir, workers, base_seed, sim_log, terminal)
}
