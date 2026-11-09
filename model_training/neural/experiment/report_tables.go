package experiment

import (
	"fmt"
	"strings"
)

func writeRegimens(text *strings.Builder, ranking []Regimen) {
	fmt.Fprint(text, "## Configurations ranked on validation games\n\nMargins are normalized score minus mean opponent score. SD is across initialization/shuffle seeds.\n\n| Configuration | Seeds | Margin | SD | Win share | Teacher KL | Agreement |\n|---|---:|---:|---:|---:|---:|---:|\n")
	for _, r := range ranking {
		fmt.Fprintf(text, "| %s | %d | %.4f | %.4f | %.1f%% | %.4f | %.1f%% |\n", r.ID, r.Seeds, r.Margin, r.MarginSD, 100*r.WinShare, r.TeacherKL, 100*r.Accuracy)
	}
	fmt.Fprint(text, "\nThe top three configurations use their seed-1 checkpoints in the held-out arena and tournament. Selection uses validation games only.\n\n")
}

func writeEvaluationTable(text *strings.Builder, title string, evaluations []Evaluation) {
	fmt.Fprintf(text, "## %s\n\n| Agent | Games | Win share | Above chance | Score margin |\n|---|---:|---:|---:|---:|\n", title)
	for _, m := range evaluations {
		if m.Group == "all" {
			fmt.Fprintf(text, "| %s | %d | %.1f%% | %+.1f pp | %+.4f |\n", m.Agent, m.Games, 100*m.WinShare, 100*m.WinAboveChance, m.Margin)
		}
	}
	fmt.Fprint(text, "\n")
}
