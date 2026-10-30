package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	rdebug "runtime/debug"

	"github.com/dgray001/gray_online/game/games/risq"
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"github.com/dgray001/gray_online/util"
)

func main() {
	if os.Getenv("GOGC") == "" {
		rdebug.SetGCPercent(400)
	}
	debug := flag.Bool("debug", false, "write per-game debug logs")
	quiet := flag.Bool("quiet", false, "discard engine stdout and skip the per-game result files; keep only errors, the sim's own per-game log and results.json")
	seed_override := flag.Int64("seed", 0, "override the scenario's base seed")
	parallel := flag.Int("parallel", runtime.NumCPU(), "games to run concurrently")
	config_dir := flag.String("config", "../config", "risq config directory")
	log_steps := flag.Bool("log-steps", false, "with -debug, also log every AI rule's select and loop step counts each turn")
	flag.Parse()
	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: sim [-config DIR] [-seed N] [-debug] [-log-steps] [-quiet] [-parallel N] <scenario_name>")
		os.Exit(1)
	}
	ai.LogRuleSteps = *log_steps
	name := flag.Arg(0)
	// the summary goes to the terminal even when engine output is redirected
	terminal := os.Stdout
	if err := risq.LoadConfig(*config_dir); err != nil {
		log.Fatalf("loading risq config: %v", err)
	}

	scenario, err := loadScenario(filepath.Join("inputs", name+".json"))
	if err != nil {
		log.Fatalf("loading scenario: %v", err)
	}

	iterations := scenario.Iterations
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

	var game_log_file *os.File
	var simStd *util.Logger
	if *quiet {
		simStd = util.NewLogger(io.Discard, util.LevelInfo)
	} else {
		// game.log: the engine's own normal output, only when not -quiet.
		var err error
		game_log_file, err = os.Create(filepath.Join(out_dir, "game.log"))
		if err != nil {
			log.Fatalf("creating game.log: %v", err)
		}
		defer game_log_file.Close()
		simStd = util.NewLogger(game_log_file, util.LevelInfo)
	}

	// errors always go to sim.log, plus game.log too when it exists
	stderr_cleanup, err := util.RedirectStderr(sim_log_file, game_log_file)
	if err != nil {
		log.Fatalf("redirecting stderr: %v", err)
	}
	defer stderr_cleanup()

	workers := max(1, *parallel)
	// a plain scenario is one sim on its map; a suite lists several, each on its own map
	sims := scenario.Buckets
	games_per_sim := scenario.GamesPerBucket
	if len(sims) == 0 {
		sims = []SuiteBucket{{Name: name, Map: scenario.Map}}
		games_per_sim = iterations
	}
	runSims(scenario, sims, max(1, games_per_sim), out_dir, workers, base_seed, *debug, *quiet, sim_log, terminal, simStd, util.DefaultErr)
}
