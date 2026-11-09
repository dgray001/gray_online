package rl

import (
	"fiddlesticks.live/neural"
	"fmt"
	"path/filepath"
)

type benchmarkAgent struct {
	Name        string
	Model       uint8
	Network     *neural.Network
	ZeroBid     bool
	RandomCards bool
}
type benchmarkResult struct {
	Evaluation Evaluation
	Policy     PolicyAudit
}

func comparisonAgents(directory string, options Options, summary savedSummary) ([]benchmarkAgent, error) {
	agents := []benchmarkAgent{}
	for id := uint8(0); id <= 4; id++ {
		agents = append(agents, benchmarkAgent{Name: fmt.Sprintf("Model %d", id), Model: id})
	}
	for _, row := range summary.Widths {
		n, err := neural.Load(filepath.Join(directory, row.Recipe+"-s1.json"))
		if err != nil {
			return nil, err
		}
		agents = append(agents, benchmarkAgent{Name: fmt.Sprintf("NN h%d", row.Hidden), Model: 4, Network: n})
	}
	if options.ReferenceHidden4 != "" {
		n, err := neural.Load(options.ReferenceHidden4)
		if err != nil {
			return nil, err
		}
		agents = append(agents, benchmarkAgent{Name: "NN h4", Model: 4, Network: n})
	}
	return comparisonAblations(agents, summary), nil
}
