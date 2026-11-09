package neural

import (
	"math"
	"testing"
)

func checkPolicyGradients(t *testing.T, n *Network, step PolicyStep, groups [][][]float64, trained [][][]float64) {
	t.Helper()
	for group, rows := range groups {
		for row, values := range rows {
			for column, value := range values {
				values[column] = value + 1e-5
				plus := policyLoss(n, step)
				values[column] = value - 1e-5
				minus := policyLoss(n, step)
				values[column] = value
				if math.Abs((plus-minus)/2e-5-(value-trained[group][row][column])) > 1e-6 {
					t.Fatalf("wrong policy gradient %d/%d/%d", group, row, column)
				}
			}
		}
	}
}
