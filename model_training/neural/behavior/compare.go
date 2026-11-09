package behavior

import (
	"fiddlesticks.live/neural"
	"math"
)

func compare(a string, b string, first []Prediction, second []Prediction, examples []neural.Example) Pair {
	pair := Pair{A: a, B: b}
	for i, example := range examples {
		x, y, tv := first[i], second[i], 0.0
		for j, value := range x.Probabilities {
			tv += math.Abs(value-y.Probabilities[j]) / 2
		}
		if example.Legal[0] < 27 {
			pair.BidCount++
			pair.BidMAE += math.Abs(x.MeanBid - y.MeanBid)
			pair.BidTV += tv
			if x.Choice != y.Choice {
				pair.BidDisagreement++
			}
		} else {
			pair.CardCount++
			pair.CardTV += tv
			if x.Choice != y.Choice {
				pair.CardDisagreement++
			}
		}
	}
	return average(pair)
}
