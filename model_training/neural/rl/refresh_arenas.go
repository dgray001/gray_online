package rl

import (
	"fmt"
	"os"
	"path/filepath"
)

func refreshArenas(options Options, results []Result, summary savedSummary, suffix string) error {
	var err error
	summary.Mixed, err = reloadArena(options.Output, "mixed-games.json", summary.Mixed)
	if err != nil {
		return err
	}
	summary.Theory4, err = reloadArena(options.Output, "theory4-games.json", summary.Theory4)
	if err != nil {
		return err
	}
	summary.Historical, err = reloadArena(options.Output, "historical-games.json", summary.Historical)
	if err != nil {
		return err
	}
	summary.Widths = widthResults(summary.Regimens, summary.Mixed, summary.Historical)
	if err := writeJSON(filepath.Join(options.Output, "summary.json"), summary); err != nil {
		return err
	}
	if err := report(options, results, summary.Regimens, summary.Mixed, summary.Theory4, summary.Comparisons, summary.Historical); err != nil {
		return err
	}
	path := filepath.Join(options.Output, "report.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, append(data, []byte(suffix)...), 0600); err != nil {
		return err
	}
	fmt.Printf("refreshed score columns: %s\n", path)
	return nil
}
