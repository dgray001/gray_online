package neural

import (
	"github.com/dgray001/gray_online/game/games/fiddlesticks"
	"math"
)

func profileFeatures(input []float64, p fiddlesticks.PublicProfile) {
	input[0] = p.BidFraction / float64(max(1, p.Bids))
	input[1] = float64(p.Exact) / float64(max(1, p.Hands))
	input[2] = p.TrickFraction / float64(max(1, p.Hands))
	input[3] = float64(p.TrumpCards) / float64(max(1, p.Cards))
	input[4] = float64(p.TrumpLeads) / float64(max(1, p.Leads))
	input[5] = float64(p.HighCards) / float64(max(1, p.Cards))
	input[6] = math.Log1p(float64(p.Hands)) / math.Log(1001)
	input[7] = math.Log1p(float64(p.Cards)) / math.Log(10001)
}
