package rl

type Metrics struct {
	Decisions         int
	Games             int
	Rounds            int
	Exact             int
	ZeroBids          int
	OpponentExact     int
	OpponentRounds    int
	ExactRate         float64
	OpponentExactRate float64
	WinShare          float64
	Margin            float64
	Score             float64
	Points            float64
	OpponentPoints    float64
	OpponentScore     float64
	MeanBid           float64
	ZeroBidRate       float64
}

type Evaluation struct {
	Hands     map[uint8]RoundMetrics
	CoreHands map[uint8]RoundMetrics
	Upper     RoundMetrics
	CoreUpper RoundMetrics
	Name      string
	All       Metrics
	Groups    map[string]Metrics
}
type Epoch struct {
	TrainingCore   Metrics
	ValidationCore Metrics
	Number         int
	Training       Metrics
	Validation     Metrics
}
type Result struct {
	InitialCore Metrics
	Trial       Trial
	Model       string
	Epochs      int
	BestEpoch   int
	Stop        string
	Initial     Metrics
	Validation  Evaluation
	Curve       []Epoch
}
