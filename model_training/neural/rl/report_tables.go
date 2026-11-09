package rl

import (
	"fmt"
	"strings"
)

func writeRanking(text *strings.Builder, ranking []Regimen) {
	fmt.Fprint(text, "## Validation ranking on 4–8 players\n\nConfigurations average their training seeds. The top three per architecture use seed 1 where available for final tests.\n\n| Configuration | Hidden | Parameters | Seeds | Initial core exact | Best core exact | Seed SD | Core win share | Core margin |\n|---|---:|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, r := range ranking {
		fmt.Fprintf(text, "| %s | %d | %d | %d | %.1f%% | %.1f%% | %.2f pp | %.1f%% | %+.4f |\n", r.ID, r.Hidden, parameterCountFor(r.Inputs, r.Hidden), r.Seeds, 100*r.Initial, 100*r.Exact, 100*r.ExactSD, 100*r.WinShare, r.Margin)
	}
	fmt.Fprint(text, "\n")
}

func writeEvaluations(text *strings.Builder, title string, evaluations []Evaluation) {
	fmt.Fprintf(text, "## %s\n\n| Agent | Games | All exact | Core exact | Core upper-half exact | Core win share | Core margin | Core zero bids |\n|---|---:|---:|---:|---:|---:|---:|---:|\n", title)
	for _, e := range evaluations {
		m := e.All
		core := selectionMetrics(e)
		fmt.Fprintf(text, "| %s | %d | %.1f%% | %.1f%% | %.1f%% | %.1f%% | %+.4f | %.1f%% |\n", e.Name, m.Games, 100*m.ExactRate, 100*core.ExactRate, 100*e.CoreUpper.ExactRate, 100*core.WinShare, core.Margin, 100*core.ZeroBidRate)
	}
	fmt.Fprint(text, "\n")
	writeArenaScores(text, evaluations)
}
