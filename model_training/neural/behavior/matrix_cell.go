package behavior

import (
	"fmt"
	"strings"
)

func matrixCell(text *strings.Builder, pair Pair, bids bool, maximum float64) {
	value := pair.CardTV
	if bids {
		value = pair.BidMAE
	}
	lightness := 100.0
	if maximum > 0 {
		lightness -= 45 * value / maximum
	}
	fmt.Fprintf(text, "<td style=\"background:hsl(210 70%% %.1f%%)\" title=\"bid MAE %.6f; bid TV %.6f; card TV %.6f; parameter RMS %s\">%.3f</td>", lightness, pair.BidMAE, pair.BidTV, pair.CardTV, weightText(pair), value)
}
