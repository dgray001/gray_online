package behavior

func average(pair Pair) Pair {
	bids, cards := float64(max(1, pair.BidCount)), float64(max(1, pair.CardCount))
	pair.BidMAE /= bids
	pair.BidTV /= bids
	pair.BidDisagreement /= bids
	pair.CardTV /= cards
	pair.CardDisagreement /= cards
	return pair
}
