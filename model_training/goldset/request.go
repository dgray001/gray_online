package goldset

type GenerateRequest struct {
	Count       int    `json:"count"`
	Players     int    `json:"players"`
	MinHand     uint8  `json:"min_hand"`
	MaxHand     uint8  `json:"max_hand"`
	Kind        string `json:"kind"`
	RoundPoints uint16 `json:"round_points"`
	TrickPoints uint16 `json:"trick_points"`
	Seed        int64  `json:"seed"`
}
