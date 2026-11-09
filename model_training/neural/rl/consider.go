package rl

func (f *fitting) consider(validation Evaluation, epoch int) bool {
	metric := selectionMetrics(validation).ExactRate
	improved := metric > selectionMetrics(f.result.Validation).ExactRate
	if improved {
		f.best = f.network.Clone()
		f.result.BestEpoch = epoch
		f.result.Validation = validation
	}
	if metric > f.reference+0.002 {
		f.reference, f.stale = metric, 0
	} else {
		f.stale++
	}
	return improved
}
