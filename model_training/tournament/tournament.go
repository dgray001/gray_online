package tournament

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/fiddlesticks"
)

type Config struct {
	Name        string     `json:"name"`
	Iterations  int        `json:"iterations"`
	MinPlayers  int        `json:"min_players"`
	MaxPlayers  int        `json:"max_players"`
	MinRound    uint8      `json:"min_round"`
	MaxRound    uint8      `json:"max_round"`
	RoundPoints uint16     `json:"round_points"`
	TrickPoints uint16     `json:"trick_points"`
	Models      []ModelDef `json:"models"`
	OutputDir   string     `json:"output_dir"`
}

type ModelDef struct {
	ID        int               `json:"id"`
	Name      string            `json:"name"`
	AIModelID uint8             `json:"ai_model_id"`
	Params    map[string]string `json:"params"`
}

type MatchStats struct {
	TotalGames      int     `json:"total_games"`
	TotalWins       int     `json:"total_wins"`
	WinRate         float64 `json:"win_rate"`
	TotalScore      float64 `json:"total_score"`
	AverageScore    float64 `json:"average_score"`
	NormalizedScore float64 `json:"normalized_score"`
	FinalElo        float64 `json:"final_elo"`
}

type TournamentResult struct {
	Timestamp   string                        `json:"timestamp"`
	Config      Config                        `json:"config"`
	Leaderboard map[string]*MatchStats        `json:"leaderboard"`
	Matrix      map[string]map[string]float64 `json:"head_to_head_winrates"`
}

type TournamentRunner struct {
	cfg      Config
	ratings  map[string]*PlayerRating
	stats    map[string]*MatchStats
	h2hWins  map[string]map[string]int
	h2hGames map[string]map[string]int
	rng      *rand.Rand
}

func NewTournamentRunner(cfg Config) *TournamentRunner {
	tr := &TournamentRunner{
		cfg:      cfg,
		ratings:  make(map[string]*PlayerRating),
		stats:    make(map[string]*MatchStats),
		h2hWins:  make(map[string]map[string]int),
		h2hGames: make(map[string]map[string]int),
		rng:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	for _, m := range cfg.Models {
		tr.ratings[m.Name] = NewPlayerRating(m.Name, 1500.0)
		tr.stats[m.Name] = &MatchStats{}
		tr.h2hWins[m.Name] = make(map[string]int)
		tr.h2hGames[m.Name] = make(map[string]int)
	}
	return tr
}

func (t *TournamentRunner) Run() (*TournamentResult, error) {
	fmt.Printf("=== Starting Tournament: %s ===\n", t.cfg.Name)
	fmt.Printf("Iterations per configuration: %d | Players: %d to %d\n", t.cfg.Iterations, t.cfg.MinPlayers, t.cfg.MaxPlayers)

	numModels := len(t.cfg.Models)
	if numModels < 2 {
		return nil, fmt.Errorf("at least 2 models required for a tournament")
	}

	for playerCount := t.cfg.MinPlayers; playerCount <= t.cfg.MaxPlayers; playerCount++ {
		fmt.Printf("Simulating %d-player tables across rounds %d to %d...\n", playerCount, t.cfg.MinRound, t.cfg.MaxRound)
		for r := t.cfg.MinRound; r <= t.cfg.MaxRound; r++ {
			if int(r)*playerCount > 51 {
				continue
			}

			for it := 0; it < t.cfg.Iterations; it++ {
				chosenIndices := t.samplePlayers(playerCount, numModels)
				t.playGame(chosenIndices, r)
			}
		}
	}

	res := t.compileResults()
	if t.cfg.OutputDir != "" {
		_ = os.MkdirAll(t.cfg.OutputDir, 0755)
		outPath := filepath.Join(t.cfg.OutputDir, t.cfg.Name+".json")
		b, _ := json.MarshalIndent(res, "", "  ")
		_ = os.WriteFile(outPath, b, 0644)
		fmt.Printf("Saved tournament results to %s\n", outPath)
	}

	t.printSummary(res)
	return res, nil
}

func (t *TournamentRunner) samplePlayers(count, poolSize int) []int {
	if count <= poolSize {
		perm := t.rng.Perm(poolSize)
		return perm[:count]
	}
	res := make([]int, count)
	for i := 0; i < count; i++ {
		res[i] = t.rng.Intn(poolSize)
	}
	return res
}

func (t *TournamentRunner) playGame(modelIndices []int, round uint8) {
	n := len(modelIndices)
	names := make([]string, n)

	gameBase := &game.GameBase{
		GameEndedChannel: make(chan string),
	}
	f := fiddlesticks.InitializeGame(gameBase)

	for i, idx := range modelIndices {
		mDef := t.cfg.Models[idx]
		names[i] = mDef.Name
		f.AddInternalAiPlayer(mDef.AIModelID, mDef.Params)
	}

	f.SetSettings(round, round, t.cfg.RoundPoints, t.cfg.TrickPoints, 1)

	go func() {
		f.GetBase().StartGame()
		f.StartAiGame(true)
		for {
			if f.ExecuteAiTurn(false) {
				break
			}
		}
	}()

	<-f.GetBase().GameEndedChannel

	maxPossible, scores := f.GetGameResults()
	floatScores := make([]float64, n)
	ratingSlice := make([]*PlayerRating, n)

	for i := 0; i < n; i++ {
		floatScores[i] = float64(scores[i])
		ratingSlice[i] = t.ratings[names[i]]

		st := t.stats[names[i]]
		st.TotalGames++
		st.TotalScore += floatScores[i]
		if maxPossible > 0 {
			st.NormalizedScore += floatScores[i] / float64(maxPossible)
		}
	}

	UpdateRatings(ratingSlice, floatScores, 24.0)

	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if i == j {
				continue
			}
			t.h2hGames[names[i]][names[j]]++
			if floatScores[i] > floatScores[j] {
				t.h2hWins[names[i]][names[j]]++
			}
		}
	}
}

func (t *TournamentRunner) compileResults() *TournamentResult {
	leaderboard := make(map[string]*MatchStats)
	h2hMatrix := make(map[string]map[string]float64)

	for name, st := range t.stats {
		st.FinalElo = t.ratings[name].Rating
		if st.TotalGames > 0 {
			st.AverageScore = st.TotalScore / float64(st.TotalGames)
			st.NormalizedScore = st.NormalizedScore / float64(st.TotalGames)
			st.WinRate = float64(t.ratings[name].Wins) / float64(st.TotalGames)
		}
		leaderboard[name] = st

		h2hMatrix[name] = make(map[string]float64)
		for opp, total := range t.h2hGames[name] {
			if total > 0 {
				h2hMatrix[name][opp] = float64(t.h2hWins[name][opp]) / float64(total)
			}
		}
	}

	return &TournamentResult{
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Config:      t.cfg,
		Leaderboard: leaderboard,
		Matrix:      h2hMatrix,
	}
}

func (t *TournamentRunner) printSummary(res *TournamentResult) {
	type entry struct {
		name string
		elo  float64
		wins int
		rate float64
		avg  float64
	}
	var list []entry
	for name, st := range res.Leaderboard {
		list = append(list, entry{
			name: name,
			elo:  st.FinalElo,
			wins: t.ratings[name].Wins,
			rate: st.WinRate * 100.0,
			avg:  st.AverageScore,
		})
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].elo > list[j].elo
	})

	fmt.Println("\n=======================================================")
	fmt.Printf(" TOURNAMENT RESULTS: %s\n", res.Config.Name)
	fmt.Println("=======================================================")
	fmt.Printf("%-14s | %-8s | %-10s | %-10s | %-10s\n", "Model", "Elo", "Win Rate", "Wins", "Avg Score")
	fmt.Println("-------------------------------------------------------")
	for _, e := range list {
		fmt.Printf("%-14s | %8.1f | %9.2f%% | %10d | %10.2f\n", e.name, e.elo, e.rate, e.wins, e.avg)
	}
	fmt.Println("=======================================================\n")
}
