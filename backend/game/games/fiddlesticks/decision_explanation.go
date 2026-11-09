package fiddlesticks

import "github.com/gin-gonic/gin"

type ExplainedCard struct {
	Index          int
	Suit           uint8
	Number         uint8
	WinProbability float64
	Contribution   float64
	Utility        float64
	Heuristic      float64
	Probability    float64
}
type DecisionExplanation struct {
	Kind               string
	RawBid             float64
	BidProbabilities   []float64
	TrickProbabilities []float64
	Cards              []ExplainedCard
	Assumption         string
	UnknownCards       int
	BeatingCards       int
	Scale              float64
	Optimism           float64
	Temperature        float64
}

func (e DecisionExplanation) ToFrontend() gin.H {
	cards := []gin.H{}
	for _, c := range e.Cards {
		cards = append(cards, gin.H{"index": c.Index, "suit": c.Suit, "number": c.Number, "win_probability": c.WinProbability, "contribution": c.Contribution, "utility": c.Utility, "heuristic": c.Heuristic, "probability": c.Probability})
	}
	return gin.H{"kind": e.Kind, "raw_bid": e.RawBid, "bid_probabilities": e.BidProbabilities, "trick_probabilities": e.TrickProbabilities, "cards": cards, "assumption": e.Assumption, "unknown_cards": e.UnknownCards, "beating_cards": e.BeatingCards, "scale": e.Scale, "optimism": e.Optimism, "temperature": e.Temperature}
}

func (m *FiddlesticksAiModelTheory4) ModelParameters() gin.H { return m.recordingParameters() }

func (m *FiddlesticksAiModelTheory4) ExplainDecision(p *FiddlesticksPlayer, f *GameFiddlesticks) DecisionExplanation {
	if f.betting {
		return m.explainBid(p, f)
	}
	return m.explainPlay(p, f)
}
