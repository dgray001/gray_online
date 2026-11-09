package rl

import (
	"fiddlesticks.live/neural"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func report(options Options, results []Result, ranking []Regimen, mixed []Evaluation, direct []Evaluation, comparisons []Comparison, historical []Evaluation) error {
	var text strings.Builder
	inputs := neural.Inputs
	if len(results) > 0 {
		inputs = results[0].Trial.Inputs
		if inputs == 0 {
			inputs = neural.LegacyInputs
		}
	}
	architecture := fmt.Sprintf("%d inputs, %d hidden neurons, 79 categorical outputs; %d policy parameters", inputs, options.Hidden, parameterCountFor(inputs, options.Hidden))
	if len(options.Widths) > 0 {
		architecture = fmt.Sprintf("%d inputs, single-hidden-layer widths %v, 79 categorical outputs; parameter counts are listed by width", inputs, options.Widths)
	}
	fmt.Fprintf(&text, "# Outcome-trained Fiddlesticks comparison\n\n%s. Network math uses only Go's standard library. All training starts from fresh weights.\n\nReward is +1 for reaching the bid and 0 for missing. Gradients credit all nonforced decisions in the round using the frozen behavior policy. Theory4 supplies opponents and benchmarks only.\n\n", architecture)
	fmt.Fprintf(&text, "Each epoch generates %d full-arc games over 2–10 players, targeting %.0f%% of games at 4–8 players. Every game reaches floor(51/players) cards and returns to one card. Five scoring presets and uniform/Theory2/Theory4/mixed/past-self opponents are covered. Validation uses %d fixed balanced-count games; 4–8-player exact-bid success selects checkpoints. Patience is %d with minimum reference improvement 0.2 percentage points.\n\n", options.GamesPerEpoch, 100*options.CoreShare, options.EvaluationGames, options.Patience)
	writeTrainingMix(&text, results)
	writeWidths(&text, widthResults(ranking, mixed, historical))
	writeWidthScores(&text, widthResults(ranking, mixed, historical))
	writeRanking(&text, ranking)
	writeEvaluations(&text, "Held-out mixed opponents", mixed)
	writeEvaluations(&text, "Held-out games against only Theory4 opponents", direct)
	writeComparisons(&text, comparisons)
	writeEvaluations(&text, "Historical 800-game short-round benchmark", historical)
	if err := handReport(options.Output, mixed); err != nil {
		return err
	}
	fmt.Fprint(&text, "[Per-hand-size results](hand-results.md). Core means 4–8 players. Upper-half means hand size at least ceil(maximum hand size / 2) for that player count. The historical arena reuses the original 45% benchmark's exact mixed-opponent deals; new full-arc percentages are not directly comparable with that old aggregate.\n\n")
	writeScope(&text, results, options)
	return os.WriteFile(filepath.Join(options.Output, "report.md"), []byte(text.String()), 0600)
}
