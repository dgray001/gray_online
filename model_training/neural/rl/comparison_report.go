package rl

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func comparisonReport(output string, results []benchmarkResult) error {
	var text strings.Builder
	fmt.Fprint(&text, "# Native models and neural-policy comparison\n\nSame 720 saved mixed-opponent full-arc games; the table uses the 400 games with 4–8 players. Scores average identical scoring presets. Win share splits ties. Model 0 uses its actual continuous random bid plus stochastic rounding; Uniform policy instead samples integer bids uniformly.\n\n| Agent | Exact bids | Mean points | Score / max | Win share | Zero bids | Card entropy / uniform | Card TV from uniform |\n|---|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, r := range results {
		m, cards := selectionMetrics(r.Evaluation), r.Policy.Cards
		fmt.Fprintf(&text, "| %s | %.2f%% | %.2f | %.2f%% | %.2f%% | %.2f%% | %.4f | %.4f |\n", r.Evaluation.Name, 100*m.ExactRate, m.Points, 100*m.Score, 100*m.WinShare, 100*m.ZeroBidRate, cards.NormalizedEntropy, cards.UniformTV)
	}
	fmt.Fprint(&text, "\nCard diagnostics include only decisions with multiple legal cards. Entropy 1 means uniform random choice, entropy 0 means deterministic choice. TV 0 means identical to uniform probabilities. High entropy alone does not establish bad play; the matched random-card ablations test usefulness. Neural actions are sampled from these distributions, not selected by argmax.\n\nThe random-card ablation uses the neural architecture with the highest mean validation success, selected before this comparison. Models and checkpoints are unchanged; this is evaluation only. summary.json preserves detailed metrics and games.json preserves outcomes and seeds.\n")
	return os.WriteFile(filepath.Join(output, "report.md"), []byte(text.String()), 0600)
}
