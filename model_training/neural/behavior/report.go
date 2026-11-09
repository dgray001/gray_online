package behavior

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func report(output string, summary Summary) error {
	var text strings.Builder
	fmt.Fprintf(&text, "# Standard-situation behavior comparison\n\n%d checkpoints, %d games, %d nonforced situations, %d checkpoint pairs. Every checkpoint receives identical feature vectors and legal actions.\n\n", len(summary.Trials), summary.Games, summary.Situations, len(summary.Pairs))
	fmt.Fprint(&text, "Bid MAE compares expected bids. Bid TV compares the checkpoint's actual categorical or rounded-expectation policy. Card TV is half the summed absolute difference in legal-card probabilities, from 0 (identical) to 1 (disjoint). Top-card disagreement compares most probable cards, not independently sampled moves. Parameter RMS is an unadjusted weight/bias distance.\n\n[matrix.html](matrix.html) contains the full heatmaps. corpus.json preserves the standard inputs; fingerprints.json preserves every prediction. This corpus covers teacher-generated situations under the experiment's game settings. Distances describe this corpus, not every possible situation.\n\n")
	writeGroups(&text, summary.Pairs)
	pairs := slices.Clone(summary.Pairs)
	slices.SortFunc(pairs, func(a Pair, b Pair) int {
		if a.CardTV < b.CardTV {
			return -1
		}
		if a.CardTV > b.CardTV {
			return 1
		}
		return 0
	})
	writePairs(&text, "Closest pairs by card behavior", pairs[:min(10, len(pairs))])
	slices.Reverse(pairs)
	writePairs(&text, "Most different pairs by card behavior", pairs[:min(10, len(pairs))])
	return os.WriteFile(filepath.Join(output, "report.md"), []byte(text.String()), 0600)
}
