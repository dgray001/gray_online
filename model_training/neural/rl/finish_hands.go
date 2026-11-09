package rl

func finishHands(e *Evaluation) {
	for hand, m := range e.Hands {
		e.Hands[hand] = averageHand(m)
	}
	for hand, m := range e.CoreHands {
		e.CoreHands[hand] = averageHand(m)
	}
	e.Upper, e.CoreUpper = averageHand(e.Upper), averageHand(e.CoreUpper)
}

func averageHand(m RoundMetrics) RoundMetrics {
	if m.Rounds > 0 {
		m.ExactRate = float64(m.Exact) / float64(m.Rounds)
		m.ZeroBidRate = float64(m.ZeroBids) / float64(m.Rounds)
	}
	return m
}
