package goldset

import "github.com/dgray001/gray_online/game/games/fiddlesticks"

func profileView(p fiddlesticks.PublicProfile) map[string]float64 {
	return map[string]float64{"hands": float64(p.Hands), "cards": float64(p.Cards), "mean_bid": p.BidFraction / float64(max(1, p.Bids)), "exact_rate": float64(p.Exact) / float64(max(1, p.Hands)), "mean_tricks": p.TrickFraction / float64(max(1, p.Hands)), "trump_rate": float64(p.TrumpCards) / float64(max(1, p.Cards)), "trump_lead_rate": float64(p.TrumpLeads) / float64(max(1, p.Leads))}
}
