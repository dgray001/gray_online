package experiment

import (
	"fiddlesticks.live/neural"
	"fmt"
	"math/rand"
)

func fitEpochs(network *neural.Network, best *neural.Network, result Result, data []neural.Example, validation []neural.Example, options Options, rng *rand.Rand, bestLoss float64, reference float64, stale int) (Result, *neural.Network) {
	for epoch := 1; epoch <= options.MaxEpochs; epoch++ {
		rng.Shuffle(len(data), func(i int, j int) { data[i], data[j] = data[j], data[i] })
		loss := trainEpoch(network, data, result.Trial)
		metric := measure(network, validation, result.Trial.Method).Loss
		result.Epochs = epoch
		result.Curve = append(result.Curve, Epoch{epoch, loss, metric})
		if metric < bestLoss {
			bestLoss, best, result.BestEpoch = metric, network.Clone(), epoch
		}
		if metric < reference-0.0001 {
			reference, stale = metric, 0
		} else {
			stale++
		}
		if epoch%10 == 0 {
			fmt.Printf("%s epoch=%d validation=%.5f\n", result.Trial.ID, epoch, metric)
		}
		if stale >= options.Patience {
			result.Stop = "validation plateau"
			break
		}
	}
	return result, best
}
