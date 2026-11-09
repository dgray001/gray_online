package fiddlesticks

type PublicAction struct {
	Seat       int
	RoundIndex int
	HandSize   uint8
	Trick      int
	Order      int
	Kind       string
	Suit       uint8
	Number     uint8
	Bid        uint8
}

type PublicProfile struct {
	Hands         int
	Bids          int
	BidFraction   float64
	Exact         int
	TrickFraction float64
	Cards         int
	TrumpCards    int
	Leads         int
	TrumpLeads    int
	HighCards     int
}
