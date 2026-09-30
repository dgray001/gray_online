package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// One game of a suite: which bucket, and which scenario player sat in each seat
type SuiteGame struct {
	Bucket string
	Result Result
	// Seats[s] is the index into the suite's players of whoever played seat s
	Seats []int
}

type suiteJob struct {
	bucket int
	index  int
}

func runSims(scenario Scenario, sims []SuiteBucket, n int, out_dir string, workers int, base_seed int64, sim_log *log.Logger, terminal *os.File) {
	games := make([][]SuiteGame, len(sims))
	for b := range games {
		games[b] = make([]SuiteGame, n)
	}
	jobs := make(chan suiteJob)
	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			for job := range jobs {
				bucket := sims[job.bucket]
				seats := make([]int, len(scenario.Players))
				for s := range seats {
					seats[s] = s
				}
				if scenario.SwapSeats && job.index%2 == 1 {
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
				if !scenario.FixedSeed {
					seed += int64(job.bucket)*1000 + int64(job.index)
				}
				max_turns := scenario.MaxTurns
				if bucket.MaxTurns > 0 {
					max_turns = bucket.MaxTurns
				}
				start := time.Now()
				result := RunGame(seed, players, bucket.Map, uint16(max_turns), 15*time.Minute)
				games[job.bucket][job.index] = SuiteGame{Bucket: bucket.Name, Result: result, Seats: seats}
				sim_log.Printf("%s game %d seed=%d seats=%v turns=%d duration=%s error=%q",
					bucket.Name, job.index+1, seed, seats, result.Game.TurnNumber, time.Since(start), result.Error)
			}
		})
	}
	for b := range sims {
		for i := range n {
			jobs <- suiteJob{bucket: b, index: i}
		}
	}
	close(jobs)
	wg.Wait()

	results := buildResults(scenario, sims, n, games)
	writeJSON(filepath.Join(out_dir, "results.json"), results)
	text := results.String()
	if err := os.WriteFile(filepath.Join(out_dir, "report.txt"), []byte(text), 0644); err != nil {
		log.Fatalf("writing report.txt: %v", err)
	}
	fmt.Fprint(terminal, text)
	sim_log.Printf("ran %d game(s) over %d sim(s), results in %s", n*len(sims), len(sims), out_dir)
}

func writeJSON(path string, v any) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		log.Fatalf("creating %s: %v", filepath.Dir(path), err)
	}
	f, err := os.Create(path)
	if err != nil {
		log.Fatalf("creating %s: %v", path, err)
	}
	defer f.Close()
	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(v); err != nil {
		log.Fatalf("writing %s: %v", path, err)
	}
}

// How one of the suite's players did over a set of games
type PlayerStats struct {
	Name  string
	Games int
	// crushed the opponent / was crushed / neither, but ahead on units at the end / neither, behind
	Crushes, Crushed, Leads, Trails int
	// mean turn of this player's crushes
	CrushTurn float64
	Errors    int
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

// One sim of a run: its map, how each player did there, and every game
type SimResults struct {
	Name  string
	Map   string
	Stats SuiteSection
	Games []SuiteGame
}

// results.json: how each player did over every game, then each sim on its own
type Results struct {
	MaxTurns    int
	GamesPerSim int
	Overall     SuiteSection
	Sims        []SimResults
}

func buildResults(scenario Scenario, sims []SuiteBucket, n int, games [][]SuiteGame) Results {
	all := make([]SuiteGame, 0)
	results := Results{MaxTurns: scenario.MaxTurns, GamesPerSim: n}
	for b, sim := range sims {
		results.Sims = append(results.Sims, SimResults{Name: sim.Name, Map: sim.Map, Stats: section(sim.Name, scenario, games[b]), Games: games[b]})
		all = append(all, games[b]...)
	}
	results.Overall = section("overall", scenario, all)
	return results
}

func section(name string, scenario Scenario, games []SuiteGame) SuiteSection {
	stats := make([]PlayerStats, len(scenario.Players))
	crush_turns := make([]int, len(scenario.Players))
	for p := range stats {
		stats[p].Name = scenario.Players[p].AiConfig
		if names := countNames(scenario.Players); names[stats[p].Name] > 1 {
			stats[p].Name = fmt.Sprintf("%s#%d", stats[p].Name, p+1)
		}
	}
	for _, g := range games {
		r := g.Result
		for seat, p := range g.Seats {
			s := &stats[p]
			s.Games++
			if r.Error != "" || seat >= len(r.Game.Players) {
				s.Errors++
				continue
			}
			pr := r.Game.Players[seat]
			switch {
			case r.Crush == seat:
				s.Crushes++
				crush_turns[p] += int(r.Game.TurnNumber)
			case r.Crush >= 0:
				s.Crushed++
			case len(r.Timeline) > 0 && finalUnits(r, seat) > finalUnits(r, 1-seat):
				s.Leads++
			default:
				s.Trails++
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
	for p := range stats {
		s := &stats[p]
		played := float64(max(1, s.Games-s.Errors))
		if s.Crushes > 0 {
			s.CrushTurn = float64(crush_turns[p]) / float64(s.Crushes)
		}
		for _, v := range []*float64{&s.Kills, &s.Lost, &s.Razes, &s.BuildingsLost, &s.Villagers, &s.Military,
			&s.VillagerIdlePct, &s.VillagerIdlePctEarly, &s.VillageCenterIdlePct, &s.VCIdlePctEarly, &s.Gathered, &s.Techs} {
			*v /= played
		}
	}
	return SuiteSection{Name: name, Players: stats}
}

func countNames(players []ScenarioPlayer) map[string]int {
	names := make(map[string]int)
	for _, p := range players {
		names[p.AiConfig]++
	}
	return names
}

// all units a seat has at the last snapshot
func finalUnits(r Result, seat int) int {
	last := r.Timeline[len(r.Timeline)-1]
	if seat < 0 || seat >= len(last) {
		return 0
	}
	total := 0
	for _, n := range last[seat].Units {
		total += n
	}
	return total
}

func (r Results) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d sim(s), %d games each, up to %d turns\n\n", len(r.Sims), r.GamesPerSim, r.MaxTurns)
	sections := []SuiteSection{r.Overall}
	for _, sim := range r.Sims {
		sections = append(sections, sim.Stats)
	}
	fmt.Fprintf(&b, "%-14s %-8s %5s %6s %7s %5s %6s %6s %6s %6s %5s %5s %6s %6s %6s %6s %8s\n",
		"sim", "ai", "games", "crush", "crushed", "lead", "trail", "crushT", "kills", "lost", "vils", "mil", "vIdle%", "vcIdle", "vcEarly", "razes", "gathered")
	for _, s := range sections {
		for _, p := range s.Players {
			fmt.Fprintf(&b, "%-14s %-8s %5d %6d %7d %5d %6d %6.0f %6.1f %6.1f %5.1f %5.1f %6.1f %6.1f %6.1f %6.1f %8.0f\n",
				s.Name, p.Name, p.Games, p.Crushes, p.Crushed, p.Leads, p.Trails, p.CrushTurn, p.Kills, p.Lost,
				p.Villagers, p.Military, p.VillagerIdlePct, p.VillageCenterIdlePct, p.VCIdlePctEarly, p.Razes, p.Gathered)
		}
		if s.Name == "overall" {
			b.WriteString("\n")
		}
	}
	if len(r.Overall.Players) == 2 {
		a, o := r.Overall.Players[0], r.Overall.Players[1]
		fmt.Fprintf(&b, "\nverdict: %s crushed %s in %d/%d games (%.0f%%), was crushed in %d, led at the end of %d more\n",
			a.Name, o.Name, a.Crushes, a.Games, 100*float64(a.Crushes)/float64(max(1, a.Games)), a.Crushed, a.Leads)
	}
	return b.String()
}
