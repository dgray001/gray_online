package behavior

import "fmt"

func weightText(pair Pair) string {
	if !pair.ParameterComparable {
		return "n/a"
	}
	return fmt.Sprintf("%.4f", pair.ParameterRMS)
}
