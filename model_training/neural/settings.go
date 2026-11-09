package neural

type GameSettings struct {
	Players        int
	MaxRound       uint8
	RoundPoints    uint16
	TrickPoints    uint16
	OpponentModels []uint8
	DealSeed       int64
	Networks       []*Network `json:"-"`
}
