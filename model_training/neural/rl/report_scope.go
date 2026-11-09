package rl

import (
	"fmt"
	"strings"
)

func writeComparisons(text *strings.Builder, comparisons []Comparison) {
	fmt.Fprint(text, "## Paired differences from Theory4 in the direct arena\n\nThe 95% intervals resample matched whole games, 2,000 bootstrap draws. Positive exact-bid delta favors RL.\n\n| Agent | Exact-bid delta | 95% interval | Win-share delta | Score-margin delta |\n|---|---:|---:|---:|---:|\n")
	for _, c := range comparisons {
		fmt.Fprintf(text, "| %s | %+.2f pp | [%+.2f, %+.2f] pp | %+.2f pp | %+.4f |\n", c.Agent, 100*c.ExactDelta, 100*c.Low, 100*c.High, 100*c.WinDelta, c.MarginDelta)
	}
	fmt.Fprint(text, "\n")
}

func writeScope(text *strings.Builder, results []Result, options Options) {
	plateaus := 0
	for _, r := range results {
		if r.Stop == "validation plateau" {
			plateaus++
		}
	}
	fmt.Fprintf(text, "## Scope\n\n%d runs plateaued; %d reached the %d-epoch cap. Fresh games mean a plateau is an early-stopping signal, not proof of optimality.\n\nExact-bid reward is not identical to maximizing game points or wins. Scoring and zero-bid statistics are reported separately. Final-test deals are disjoint from training and validation. manifest.json records source hash, reward, policy type, seeds, and options; trials.json records learning curves; arena files preserve completed bids, tricks, and points.\n\nMethod reference: [Sutton and Barto, policy-gradient methods](https://incompleteideas.net/book/bookdraft2018mar21.pdf).\n", plateaus, len(results)-plateaus, options.MaxEpochs)
}
