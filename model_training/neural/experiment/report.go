package experiment

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func report(options Options, results []Result, ranking []Regimen, test []Evaluation, tournament []Evaluation) error {
	var text strings.Builder
	fmt.Fprintf(&text, "# Linear Fiddlesticks training experiment\n\n370 inputs, 79 outputs, no hidden layer; 29,309 parameters. All network math uses Go's standard library.\n\n%d training games, %d validation games, %d held-out games. Whole games use disjoint split seeds. Forced moves are excluded from prediction metrics.\n\n", options.TrainGames, options.ValidationGames, options.TestGames)
	fmt.Fprintf(&text, "24 trials: hard labels or teacher distributions; zero or Xavier initialization; learning rate 0.01 or 0.05; three seeds. Early stopping uses validation loss, patience %d, minimum improvement 0.0001. Hard loss is cross-entropy; soft loss is teacher KL.\n\n", options.Patience)
	writeRegimens(&text, ranking)
	writeEvaluationTable(&text, "Held-out games against fixed opponents", test)
	writeEvaluationTable(&text, "Selected networks and Theory4 tournament", tournament)
	writeStops(&text, results, options)
	return os.WriteFile(filepath.Join(options.Output, "report.md"), []byte(text.String()), 0600)
}
