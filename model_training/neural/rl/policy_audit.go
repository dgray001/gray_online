package rl

import "math"

type ActionAudit struct {
	Decisions         int
	NormalizedEntropy float64
	UniformTV         float64
	MaxProbability    float64
	ZeroProbability   float64
}
type PolicyAudit struct {
	Bids  ActionAudit
	Cards ActionAudit
}

func (a *PolicyAudit) observe(legal []int, probabilities []float64) {
	m := &a.Cards
	if legal[0] < 27 {
		m = &a.Bids
		m.ZeroProbability += probabilities[0]
	}
	m.Decisions++
	entropy, distance, maximum := 0.0, 0.0, 0.0
	for _, action := range legal {
		p := probabilities[action]
		if p > 0 {
			entropy -= p * math.Log(p)
		}
		distance += math.Abs(p-1/float64(len(legal))) / 2
		maximum = max(maximum, p)
	}
	m.NormalizedEntropy += entropy / math.Log(float64(len(legal)))
	m.UniformTV += distance
	m.MaxProbability += maximum
}

func (a PolicyAudit) averaged() PolicyAudit {
	for _, m := range []*ActionAudit{&a.Bids, &a.Cards} {
		n := float64(max(1, m.Decisions))
		m.NormalizedEntropy /= n
		m.UniformTV /= n
		m.MaxProbability /= n
		m.ZeroProbability /= n
	}
	return a
}
