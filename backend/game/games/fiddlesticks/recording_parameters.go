package fiddlesticks

import "github.com/gin-gonic/gin"

func (m *FiddlesticksAiModelTheory4) recordingParameters() gin.H {
	return gin.H{
		"win_vs_all_trump_factor":    m.win_vs_all_trump_factor,
		"win_vs_all_nontrump_factor": m.win_vs_all_nontrump_factor,
		"min_win_weight_factor":      m.min_win_weight_factor,
		"bet_scale":                  m.bet_scale, "bet_optimism": m.bet_optimism,
		"trump_hoarding": m.trump_hoarding,
		"void_hunting":   m.void_hunting,
		"vindictiveness": m.vindictiveness,
		"bid_trust":      m.bid_trust, "bid_sigma": m.bid_sigma,
		"planner_temperature": m.temperature,
		"overtrick_aversion":  m.overtrick_aversion,
		"undertrick_aversion": m.undertrick_aversion,
	}
}
