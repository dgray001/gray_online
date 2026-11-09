package goldset

type PublicPlayer struct {
	Seat         int                `json:"seat"`
	Score        uint32             `json:"score"`
	Bid          uint8              `json:"bid"`
	HasBid       bool               `json:"has_bid"`
	Tricks       uint8              `json:"tricks"`
	Voids        [4]bool            `json:"voids"`
	Profile      map[string]float64 `json:"profile"`
	PriorProfile map[string]float64 `json:"prior_profile"`
}
type Progress struct {
	Index        int     `json:"index"`
	Total        int     `json:"total"`
	Remaining    int     `json:"remaining"`
	FuturePoints float64 `json:"future_points"`
}
type Event struct {
	Seat   int    `json:"seat"`
	Round  int    `json:"round"`
	Hand   uint8  `json:"hand"`
	Trick  int    `json:"trick"`
	Order  int    `json:"order"`
	Kind   string `json:"kind"`
	Suit   uint8  `json:"suit"`
	Number uint8  `json:"number"`
	Bid    uint8  `json:"bid"`
}
type State struct {
	Kind        string         `json:"kind"`
	Objective   string         `json:"objective"`
	Seat        int            `json:"seat"`
	Dealer      int            `json:"dealer"`
	TrickLeader int            `json:"trick_leader"`
	HandSize    uint8          `json:"hand_size"`
	MaxHand     uint8          `json:"max_hand"`
	Increasing  bool           `json:"increasing"`
	Progress    Progress       `json:"progress"`
	RoundPoints uint16         `json:"round_points"`
	TrickPoints uint16         `json:"trick_points"`
	Trump       Card           `json:"trump"`
	Hand        []Card         `json:"hand"`
	Trick       []Card         `json:"trick"`
	Players     []PublicPlayer `json:"players"`
	History     []Event        `json:"history"`
}
