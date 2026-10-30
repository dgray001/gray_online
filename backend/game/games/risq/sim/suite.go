package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/dgray001/gray_online/util"
)

// One game of a suite: which bucket, and which scenario player sat in each seat
type SuiteGame struct {
	Bucket string
	Map    string
	Result Result
	// Seats[s] is the index into the suite's players of whoever played seat s
	Seats []int
}

type suiteJob struct {
	bucket int
	index  int
}

func runSims(scenario Scenario, sims []SuiteBucket, n int, out_dir string, workers int, base_seed int64, debug bool, quiet bool, sim_log *log.Logger, terminal *os.File, std, err *util.Logger) {
	results := Results{MaxTurns: scenario.MaxTurns, GamesPerSim: n, Overall: newSection("overall", scenario)}
	for _, sim := range sims {
		results.Sims = append(results.Sims, SimResults{Name: sim.Name, Map: sim.Map, Stats: newSection(sim.Name, scenario)})
	}
	var results_mu sync.Mutex
	jobs := make(chan suiteJob)
	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			for job := range jobs {
				bucket := sims[job.bucket]
				seats := make([]int, bucket.seatCount(len(scenario.Players)))
				for s := range seats {
					seats[s] = s % len(scenario.Players)
				}
				swapped := scenario.SwapSeats && job.index%2 == 1
				if bucket.Seed != 0 {
					swapped = bucket.Swap
				}
				if swapped {
					for l, r := 0, len(seats)-1; l < r; l, r = l+1, r-1 {
						seats[l], seats[r] = seats[r], seats[l]
					}
				}
				players := make([]PlayerConfig, len(seats))
				for s, p := range seats {
					players[s] = PlayerConfig{Nickname: fmt.Sprintf("ai-%d", s), ConfigPath: scenario.Players[p].AiConfig}
				}
				// every game has its own seed, fixed by sim and game number, so any game can be replayed on its own
				seed := base_seed
				if bucket.Seed != 0 {
					seed = bucket.Seed
				} else if !scenario.FixedSeed {
					seed += int64(job.bucket)*1000 + int64(job.index)
				}
				max_turns := scenario.MaxTurns
				if bucket.MaxTurns > 0 {
					max_turns = bucket.MaxTurns
				}
				start := time.Now()
				var debug_file *os.File
				gameStd := std
				if debug {
					var err error
					name := "debug.log"
					if len(sims) > 1 || n > 1 {
						name = fmt.Sprintf("debug_%d.log", seed)
					}
					if debug_file, err = os.Create(filepath.Join(out_dir, name)); err != nil {
						log.Fatalf("creating debug log for seed %d: %v", seed, err)
					}
					gameStd = util.NewLogger(debug_file, util.LevelDebug)
				}
				result := RunGame(seed, players, seats, bucket.Map, bucket.Settings, scenario.Metrics, uint16(max_turns), 15*time.Minute, gameStd, err)
				if debug_file != nil {
					debug_file.Close()
				}
				game := SuiteGame{Bucket: bucket.Name, Map: bucket.Map, Result: result, Seats: seats}
				results_mu.Lock()
				results.Sims[job.bucket].Stats.add(game)
				results.Sims[job.bucket].Games++
				results.Overall.add(game)
				results_mu.Unlock()
				if !quiet {
					writeGameFile(out_dir, job.index, game)
				}
				sim_log.Printf("%s game %d seed=%d map=%s settings=%v seats=%v turns=%d outcomes=%v duration=%s error=%q",
					bucket.Name, job.index+1, seed, bucket.Map, bucket.Settings, seats, result.Game.TurnNumber, result.Outcomes, time.Since(start), result.Error)
			}
		})
	}
	for b := range sims {
		for i := range sims[b].gameCount(n) {
			jobs <- suiteJob{bucket: b, index: i}
		}
	}
	close(jobs)
	wg.Wait()

	results.finish()
	text := results.String()
	data, write_err := json.MarshalIndent(results, "", "  ")
	if write_err == nil {
		write_err = os.WriteFile(filepath.Join(out_dir, "results.json"), data, 0644)
	}
	if write_err != nil {
		log.Fatalf("writing results.json: %v", write_err)
	}
	fmt.Fprint(terminal, text)
	sim_log.Printf("ran %d game(s) over %d sim(s), results in %s", results.totalGames(), len(sims), out_dir)
}

func writeGameFile(out_dir string, index int, game SuiteGame) {
	dir := filepath.Join(out_dir, "games")
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatalf("creating %s: %v", dir, err)
	}
	path := filepath.Join(dir, fmt.Sprintf("%s_%d_%d.json", game.Bucket, index+1, game.Result.Seed))
	data, err := json.MarshalIndent(game, "", "  ")
	if err == nil {
		err = os.WriteFile(path, data, 0644)
	}
	if err != nil {
		log.Fatalf("writing %s: %v", path, err)
	}
}

// How one of the suite's players did over a set of games
type PlayerStats struct {
	Name  string
	Games int
	// seats of this player that finished without error; the divisor for the per-game means, since a side can hold several seats
	seats_played int
	// games ending each way from this player's own side
	Outcomes map[Outcome]int
	Errors   int
	// per-game means
	Kills, Lost, Razes, BuildingsLost     float64
	Villagers, Military                   float64
	VillagerIdlePct, VillagerIdlePctEarly float64
	VillageCenterIdlePct, VCIdlePctEarly  float64
	Gathered                              float64
	Techs                                 float64
}

type SuiteSection struct {
	Name    string
	Players []PlayerStats
}

// One sim of a run: its map, how many games it played, and how each player did there
type SimResults struct {
	Name  string
	Map   string
	Stats SuiteSection
	Games int
}

// results.json: how each player did over every game, then each sim on its own
type Results struct {
	MaxTurns    int
	GamesPerSim int
	Overall     SuiteSection
	Sims        []SimResults
}

func (r *Results) finish() {
	r.Overall.finish()
	for i := range r.Sims {
		r.Sims[i].Stats.finish()
	}
}

func newSection(name string, scenario Scenario) SuiteSection {
	stats := make([]PlayerStats, len(scenario.Players))
	for p := range stats {
		stats[p].Outcomes = make(map[Outcome]int)
		stats[p].Name = scenario.Players[p].AiConfig
		if names := countNames(scenario.Players); names[stats[p].Name] > 1 {
			stats[p].Name = fmt.Sprintf("%s#%d", stats[p].Name, p+1)
		}
	}
	return SuiteSection{Name: name, Players: stats}
}

func (section *SuiteSection) add(g SuiteGame) {
	stats := section.Players
	{
		r := g.Result
		for seat, p := range g.Seats {
			s := &stats[p]
			first_seat := seat == slices.Index(g.Seats, p)
			if first_seat {
				s.Games++
			}
			if r.Error != "" || seat >= len(r.Game.Players) || seat >= len(r.Outcomes) {
				if first_seat {
					s.Errors++
				}
				continue
			}
			pr := r.Game.Players[seat]
			s.seats_played++
			if first_seat {
				s.Outcomes[r.Outcomes[seat]]++
			}
			s.Kills += float64(pr.Kills)
			s.Lost += float64(pr.UnitsLost)
			s.Razes += float64(pr.Razes)
			s.BuildingsLost += float64(pr.BuildingsLost)
			if len(r.Timeline) > 0 {
				last := r.Timeline[len(r.Timeline)-1][seat]
				for id, count := range last.Units {
					if id == 1 {
						s.Villagers += float64(count)
					} else {
						s.Military += float64(count)
					}
				}
			}
			s.VillagerIdlePct += pr.Economy.VillagerIdlePct
			s.VillagerIdlePctEarly += pr.Economy.VillagerIdlePctEarly
			s.VillageCenterIdlePct += pr.Economy.VillageCenterIdlePct
			s.VCIdlePctEarly += pr.Economy.VillageCenterIdlePctEarly
			s.Gathered += pr.Gathered.Food + pr.Gathered.Wood + pr.Gathered.Stone + pr.Gathered.Gold
			s.Techs += float64(pr.TechsResearched)
		}
	}
}

func (section *SuiteSection) finish() {
	stats := section.Players
	for p := range stats {
		s := &stats[p]
		played := float64(max(1, s.seats_played))
		for _, v := range []*float64{&s.Kills, &s.Lost, &s.Razes, &s.BuildingsLost, &s.Villagers, &s.Military,
			&s.VillagerIdlePct, &s.VillagerIdlePctEarly, &s.VillageCenterIdlePct, &s.VCIdlePctEarly, &s.Gathered, &s.Techs} {
			*v /= played
		}
	}
}

func countNames(players []ScenarioPlayer) map[string]int {
	names := make(map[string]int)
	for _, p := range players {
		names[p.AiConfig]++
	}
	return names
}

func (r Results) totalGames() int {
	total := 0
	for _, sim := range r.Sims {
		total += sim.Games
	}
	return total
}

func (r Results) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d sim(s), %d games, up to %d turns\n\n", len(r.Sims), r.totalGames(), r.MaxTurns)
	sections := []SuiteSection{r.Overall}
	for _, sim := range r.Sims {
		sections = append(sections, sim.Stats)
	}
	fmt.Fprintf(&b, "%-14s %-8s %5s", "sim", "ai", "games")
	for _, o := range outcomeOrder {
		fmt.Fprintf(&b, " %8s", o)
	}
	fmt.Fprintf(&b, " %6s %6s %5s %5s %6s %6s %7s %6s %8s\n", "kills", "lost", "vils", "mil", "vIdle%", "vcIdle", "vcEarly", "razes", "gathered")
	for _, s := range sections {
		for _, p := range s.Players {
			fmt.Fprintf(&b, "%-14s %-8s %5d", s.Name, p.Name, p.Games)
			for _, o := range outcomeOrder {
				fmt.Fprintf(&b, " %8d", p.Outcomes[o])
			}
			fmt.Fprintf(&b, " %6.1f %6.1f %5.1f %5.1f %6.1f %6.1f %7.1f %6.1f %8.0f\n", p.Kills, p.Lost,
				p.Villagers, p.Military, p.VillagerIdlePct, p.VillageCenterIdlePct, p.VCIdlePctEarly, p.Razes, p.Gathered)
		}
		if s.Name == "overall" {
			b.WriteString("\n")
		}
	}
	if len(r.Overall.Players) == 2 {
		a := r.Overall.Players[0]
		fmt.Fprintf(&b, "\nverdict: %s over %d games:", a.Name, a.Games)
		for _, o := range outcomeOrder {
			fmt.Fprintf(&b, " %d %s,", a.Outcomes[o], o)
		}
		b.WriteString("\n")
	}
	return b.String()
}
